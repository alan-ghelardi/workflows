package workflow

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.uber.org/zap"

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
	DeployKeys github.Syncer

	// Webhook allows us to manages the state of Github Webhooks.
	Webhook github.Syncer

	// KubeClientSet allows us to talk to the k8s for core APIs.
	KubeClientSet kubernetes.Interface

	// workflowLister indexes workflow objects.
	workflowLister listers.WorkflowLister

	// WorkflowsClientSet allows us to configure Workflows objects.
	WorkflowsClientSet workflowsclientset.Interface
}

// Check that our Reconciler implements Interface
var _ workflowreconciler.Interface = (*Reconciler)(nil)

// ReconcileKind implements Interface.ReconcileKind.
func (r *Reconciler) ReconcileKind(ctx context.Context, workflow *workflowsv1alpha1.Workflow) reconciler.Event {
	logger := logging.FromContext(ctx).With("namespace", workflow.GetNamespace(), "name", workflow.GetName())
	logger.Info("Reconciling workflow")

	if err := r.reconcileWebhook(ctx, logger, workflow); err != nil {
		return err
	}

	if err := r.reconcileDeployKeys(ctx, logger, workflow); err != nil {
		return err
	}

	return nil
}

// reconcileWebhook keeps Github Webhooks in sync with the desired state
// declared in workflows.
func (r *Reconciler) reconcileWebhook(ctx context.Context, logger *zap.SugaredLogger, workflow *workflowsv1alpha1.Workflow) error {
	result, err := r.Webhook.Sync(ctx, workflow)
	if err != nil {
		logger.Error("Unable to sync Github Webhook", err)
		return err
	}

	if result.HasCreatedResources() {
		if err := r.updateAnnotations(ctx, workflow); err != nil {
			logger.Error("Unable to update workflow's annotations", err)
			return err
		}

		webhookSecret := secret.OfWebhook(workflow, result)
		if err := r.reconcileSecret(ctx, workflow, webhookSecret); err != nil {
			logger.Error("Unable to reconcile Webhook secret token", err)
			return err
		}
	}

	return nil
}

// updateAnnotations ...
func (r *Reconciler) updateAnnotations(ctx context.Context, workflow *workflowsv1alpha1.Workflow) error {
	newWorkflow, err := r.workflowLister.Workflows(workflow.GetNamespace()).Get(workflow.GetName())
	if err != nil {
		return fmt.Errorf("Failed to get workflow %s from shared cache: %w", workflow.GetName(), err)
	}

	newWorkflow.Annotations = workflow.Annotations

	if _, err := r.WorkflowsClientSet.WorkflowsV1alpha1().Workflows(workflow.GetNamespace()).Update(ctx, newWorkflow, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("Failed to update workflow %s: %w", workflow.GetName(), err)
	}

	return nil
}

// reconcileSecret creates the supplied secret object if it doesn't exist in the cluster.
func (r *Reconciler) reconcileSecret(ctx context.Context, workflow *workflowsv1alpha1.Workflow, secretObj *corev1.Secret) error {
	if err := r.KubeClientSet.CoreV1().Secrets(workflow.GetNamespace()).Delete(ctx, workflow.GetName(), metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("Unable to delete secret %s: %w", secretObj.GetName(), err)
	}

	if _, err := r.KubeClientSet.CoreV1().Secrets(workflow.GetNamespace()).Create(ctx, secretObj, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("Unable to create secret %s: %w", secretObj.GetName(), err)
	}

	if err := r.setOwnerReference(workflow, secretObj); err != nil {
		return fmt.Errorf("Unable to set owner references for secret %s: %w", secretObj.GetName(), err)
	}

	return nil
}

// setOwnerReference makes the provided object dependent on the workflow. Thus,
// if the workflow is deleted, the supplied object will be garbage collected.
func (r *Reconciler) setOwnerReference(workflow *workflowsv1alpha1.Workflow, secret *corev1.Secret) error {
	ownerRef := metav1.NewControllerRef(workflow, workflow.GetGroupVersionKind())
	references := []metav1.OwnerReference{*ownerRef}
	secret.SetOwnerReferences(references)
	return nil
}

// reconcileDeployKeys keeps Github deploy keys in sync with the desired state
// declared in workflows.
func (r *Reconciler) reconcileDeployKeys(ctx context.Context, logger *zap.SugaredLogger, workflow *workflowsv1alpha1.Workflow) error {
	result, err := r.DeployKeys.Sync(ctx, workflow)
	if err != nil {
		logger.Error("Unable to sync Github deploy keys", err)
		return err
	}

	if result.HasCreatedResources() {
		if err := r.updateAnnotations(ctx, workflow); err != nil {
			logger.Error("Unable to update workflow's annotations", err)
			return err
		}

		privateSSHKey := secret.OfDeployKeys(workflow, result)
		if err := r.reconcileSecret(ctx, workflow, privateSSHKey); err != nil {
			logger.Error("Unable to reconcile private SSH keys", err)
			return err
		}
	}

	return nil
}
