package workflow

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	"github.com/nubank/workflows/pkg/github"
	"github.com/nubank/workflows/pkg/secret"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	workflowreconciler "github.com/nubank/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	listers "github.com/nubank/workflows/pkg/client/listers/workflows/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

// Reconciler implements workflowreconciler.Interface for
// Workflow resources.
type Reconciler struct {

	// DeployKeys allow us to manage Github deploy keys.
	DeployKeys *github.DeployKeySyncer

	// Webhook allows us to manages the state of Github Webhooks.
	Webhook *github.WebhookSyncer

	// KubeClientSet allows us to talk to the k8s for core APIs.
	KubeClientSet kubernetes.Interface

	// workflowLister indexes workflow objects.
	workflowLister listers.WorkflowLister

	// WorkflowsClientSet allows us to configure Workflows objects.
	WorkflowsClientSet workflowsclientset.Interface
}

// Check that our Reconciler implements the required interfaces
var (
	_ workflowreconciler.Interface = (*Reconciler)(nil)
	_ workflowreconciler.Finalizer = (*Reconciler)(nil)
)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, workflow *workflowsv1alpha1.Workflow) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Info("Reconciling workflow")

	if err := r.reconcileWebhook(ctx, workflow); err != nil {
		workflow.Status.MarkWebhookError(err.Error())
		return err
	}

	if err := r.reconcileDeployKeys(ctx, workflow); err != nil {
		workflow.Status.MarkDeployKeysError(err.Error())
		return err
	}

	workflow.Status.MarkReady()

	return nil
}

// reconcileWebhook keeps Github Webhooks in sync with the desired state
// declared in workflows.
func (r *Reconciler) reconcileWebhook(ctx context.Context, workflow *workflowsv1alpha1.Workflow) error {
	webhook, err := r.Webhook.Sync(ctx, workflow)
	if err != nil {
		return err
	}

	if webhook != nil && len(webhook.Secret) != 0 {
		// There were changes in the Github Webhook and a new secret was
		// created. Thus, we try to reconcile the Secret object that
		// stores the Webhook secret token.
		if err := r.reconcileWebhookSecret(ctx, workflow, webhook); err != nil {
			return err
		}
	}

	return nil
}

// reconcileWebhookSecret
func (r *Reconciler) reconcileWebhookSecret(ctx context.Context, workflow *workflowsv1alpha1.Workflow, webhook *github.Webhook) error {
	var (
		webhookSecret *corev1.Secret
		err           error
	)

	webhookSecret, err = r.getSecret(ctx, workflow.GetNamespace(), workflow.GetWebhookSecretName())
	if err != nil {
		return err
	}

	if webhookSecret == nil {
		webhookSecret = secret.OfWebhook(workflow, webhook.Secret)
		err = r.createSecret(ctx, workflow, webhookSecret)
	} else {
		secret.SetSecretToken(webhookSecret, webhook.Secret)
		err = r.updateSecret(ctx, webhookSecret)
	}

	return err
}

// getSecret gets the Secret object identified by the supplied name in the
// supplied namespace or nil if it doesn't exist.
func (r *Reconciler) getSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	var (
		secret *corev1.Secret
		err    error
	)

	secret, err = r.KubeClientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("Error getting Secret %s/%s: %w", namespace, name, err)
	}

	if apierrors.IsNotFound(err) {
		return nil, nil
	}

	return secret, nil
}

// createSecret creates the supplied Secret object.
func (r *Reconciler) createSecret(ctx context.Context, workflow *workflowsv1alpha1.Workflow, secret *corev1.Secret) error {
	r.setOwnerReference(workflow, secret)

	if _, err := r.KubeClientSet.CoreV1().Secrets(secret.GetNamespace()).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("Error creating Secret %s/%s: %w", secret.GetNamespace(), secret.GetName(), err)
	}

	logger := logging.FromContext(ctx)

	logger.Infof("Successfully created Secret %s/%s", secret.GetNamespace(), secret.GetName())

	return nil
}

// setOwnerReference makes the provided Secret object dependent on the
// workflow. Thus, if the workflow is deleted, the Secret will be garbage
// collected.
func (r *Reconciler) setOwnerReference(workflow *workflowsv1alpha1.Workflow, secret *corev1.Secret) {
	ownerRef := metav1.NewControllerRef(workflow, workflow.GetGroupVersionKind())
	references := []metav1.OwnerReference{*ownerRef}
	secret.SetOwnerReferences(references)
}

// updateSecret updates the supplied Secret object.
func (r *Reconciler) updateSecret(ctx context.Context, secret *corev1.Secret) error {
	if _, err := r.KubeClientSet.CoreV1().Secrets(secret.GetNamespace()).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("Error updating Secret %s/%s: %w", secret.GetNamespace(), secret.GetName(), err)
	}

	logger := logging.FromContext(ctx)
	logger.Infof("Successfully updated Secret %s/%s", secret.GetNamespace(), secret.GetName())

	return nil
}

// reconcileDeployKeys keeps Github deploy keys in sync with the desired state
// declared in workflows.
func (r *Reconciler) reconcileDeployKeys(ctx context.Context, workflow *workflowsv1alpha1.Workflow) error {
	keyPairs, err := r.DeployKeys.Sync(ctx, workflow)
	if err != nil {
		return err
	}

	if len(keyPairs) != 0 {
		// There were changes in Github deploy keys and we should create
		// or update the Secret that stores SSH private keys.
		if err := r.reconcileDeployKeysSecret(ctx, workflow, keyPairs); err != nil {
			return err
		}
	}

	return nil
}

// reconcileDeployKeysSecret
func (r *Reconciler) reconcileDeployKeysSecret(ctx context.Context, workflow *workflowsv1alpha1.Workflow, keyPairs []secret.KeyPair) error {
	var (
		deployKeys *corev1.Secret
		err        error
	)

	deployKeys, err = r.getSecret(ctx, workflow.GetNamespace(), workflow.GetDeployKeysSecretName())
	if err != nil {
		return err
	}

	logger := logging.FromContext(ctx)

	if deployKeys == nil {
		logger.Info("Creating new Secret to store SSH private keys")
		deployKeys = secret.OfDeployKeys(workflow, keyPairs)
		err = r.createSecret(ctx, workflow, deployKeys)
	} else {
		logger.Infof("Updating Secret %s/%s with the newly-created SSH private keys", deployKeys.GetNamespace(), deployKeys.GetName())
		secret.SetSSHPrivateKeys(deployKeys, keyPairs)
		err = r.updateSecret(ctx, deployKeys)
	}

	return err
}

// FinalizeKind implements Finalizer.FinalizeKind.
func (r *Reconciler) FinalizeKind(ctx context.Context, workflow *workflowsv1alpha1.Workflow) reconciler.Event {
	if err := r.Webhook.Delete(ctx, workflow); err != nil {
		return err
	}

	if err := r.DeployKeys.Delete(ctx, workflow); err != nil {
		return err
	}
	return nil
}
