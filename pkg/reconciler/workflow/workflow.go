package workflow

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	"github.com/nubank/workflows/pkg/github"
	"github.com/nubank/workflows/pkg/secrets"

	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
	workflowsclientset "github.com/nubank/workflows/pkg/client/clientset/versioned"
	workflowreconciler "github.com/nubank/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	listers "github.com/nubank/workflows/pkg/client/listers/workflows/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/kmp"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

// Reconciler implements workflowreconciler.Interface for
// Workflow resources.
type Reconciler struct {

	// deployKeys allow us to manage Github deploy keys.
	deployKeys github.DeployKeysReconciler

	// webhook allows us to manage the state of Github Webhooks.
	webhook github.WebhookReconciler

	// repositories allows us to configure workflow's repositories according
	// to properties of Github repos.
	repositories github.RepoReconciler

	// kubeClientSet allows us to talk to the k8s for core APIs.
	kubeClientSet kubernetes.Interface

	// workflowLister indexes workflow objects.
	workflowLister listers.WorkflowLister

	// workflowsClientSet allows us to configure Workflows objects.
	workflowsClientSet workflowsclientset.Interface
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

	// Save a copy with the current state to compare later
	currentWorkflow := workflow.DeepCopy()

	// Set important attributes such as Repository.DefaultBranch and
	// Repository.Private in all repositories declared in this workflow
	if err := r.repositories.ReconcileRepos(ctx, workflow); err != nil {
		workflow.Status.MarkRepositoriesError(err.Error())
		return err
	}

	if err := r.reconcileWebhook(ctx, workflow); err != nil {
		workflow.Status.MarkWebhookError(err.Error())
		return err
	}

	if err := r.reconcileDeployKeys(ctx, workflow); err != nil {
		workflow.Status.MarkDeployKeysError(err.Error())
		return err
	}

	// Verify whether the spec has changed (e.g. repository settings were modified).
	equal, err := kmp.SafeEqual(workflow.Spec, currentWorkflow.Spec)
	if err != nil {
		return fmt.Errorf("Error diffing workflow: %w", err)
	}

	if !equal {
		// The spec has changed and we'll sync the changes.
		if _, err := r.workflowsClientSet.WorkflowsV1alpha1().Workflows(workflow.GetNamespace()).Update(ctx, workflow, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	workflow.Status.MarkReady()

	return nil
}

// reconcileWebhook keeps Github Webhooks in sync with the desired state
// declared in workflows.
func (r *Reconciler) reconcileWebhook(ctx context.Context, workflow *workflowsv1alpha1.Workflow) error {
	webhook, err := r.webhook.ReconcileHook(ctx, workflow)
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

// reconcileWebhookSecret creates or updates the corev1.Secret object that holds the Webhook secret token.
func (r *Reconciler) reconcileWebhookSecret(ctx context.Context, workflow *workflowsv1alpha1.Workflow, webhook *github.Webhook) error {
	var (
		webhookSecret *corev1.Secret
		err           error
	)

	logger := logging.FromContext(ctx)

	webhookSecret, err = r.getSecret(ctx, workflow.GetNamespace(), workflow.GetWebhookSecretName())
	if err != nil {
		return err
	}

	if webhookSecret == nil {
		webhookSecret = secrets.OfWebhook(workflow, webhook.Secret)
		logger.Info("Creating a new Secret resource to store the newly-created Webhook secret token")
		err = r.createSecret(ctx, workflow, webhookSecret)
	} else {
		logger.Infof("Updating Secret %s/%s with the newly-created Webhook secret token", webhookSecret.GetNamespace(), webhookSecret.GetName())
		secrets.SetSecretToken(webhookSecret, webhook.Secret)
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

	secret, err = r.kubeClientSet.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})

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
	if _, err := r.kubeClientSet.CoreV1().Secrets(secret.GetNamespace()).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("Error creating Secret %s/%s: %w", secret.GetNamespace(), secret.GetName(), err)
	}

	logger := logging.FromContext(ctx)
	logger.Infof("Secret %s/%s has been successfully created", secret.GetNamespace(), secret.GetName())

	return nil
}

// updateSecret updates the supplied Secret object.
func (r *Reconciler) updateSecret(ctx context.Context, secret *corev1.Secret) error {
	if _, err := r.kubeClientSet.CoreV1().Secrets(secret.GetNamespace()).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("Error updating Secret %s/%s: %w", secret.GetNamespace(), secret.GetName(), err)
	}

	logger := logging.FromContext(ctx)
	logger.Infof("Secret %s/%s has been successfully updated", secret.GetNamespace(), secret.GetName())

	return nil
}

// reconcileDeployKeys keeps Github deploy keys in sync with the desired state
// declared in workflows.
func (r *Reconciler) reconcileDeployKeys(ctx context.Context, workflow *workflowsv1alpha1.Workflow) error {
	keyPairs, err := r.deployKeys.ReconcileKeys(ctx, workflow)
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

// reconcileDeployKeysSecret creates or updates the corev1.Secret object that holds SSH private keys for the workflow in question.
func (r *Reconciler) reconcileDeployKeysSecret(ctx context.Context, workflow *workflowsv1alpha1.Workflow, keyPairs []secrets.KeyPair) error {
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
		deployKeys = secrets.OfDeployKeys(workflow, keyPairs)
		err = r.createSecret(ctx, workflow, deployKeys)
	} else {
		logger.Infof("Updating Secret %s/%s with the newly-created SSH private keys", deployKeys.GetNamespace(), deployKeys.GetName())
		secrets.SetSSHPrivateKeys(deployKeys, keyPairs)
		err = r.updateSecret(ctx, deployKeys)
	}

	return err
}

// FinalizeKind implements Finalizer.FinalizeKind.
func (r *Reconciler) FinalizeKind(ctx context.Context, workflow *workflowsv1alpha1.Workflow) reconciler.Event {
	if err := r.webhook.Delete(ctx, workflow); err != nil {
		return err
	}

	if err := r.deployKeys.Delete(ctx, workflow); err != nil {
		return err
	}
	return nil
}
