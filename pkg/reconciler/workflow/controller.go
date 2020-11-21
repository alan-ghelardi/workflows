package workflow

import (
	"context"

	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"

	workflowsclient "github.com/nubank/workflows/pkg/client/injection/client"
	workflowinformer "github.com/nubank/workflows/pkg/client/injection/informers/workflows/v1alpha1/workflow"
	workflowreconciler "github.com/nubank/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	"github.com/nubank/workflows/pkg/github"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	logger := logging.FromContext(ctx)

	workflowInformer := workflowinformer.Get(ctx)

	reconciler := &Reconciler{DeployKeys: github.GetDeployKeySyncerOrDie(ctx),
		Webhook:            github.GetWebhookSyncerOrDie(ctx),
		KubeClientSet:      kubeclient.Get(ctx),
		WorkflowsClientSet: workflowsclient.Get(ctx),
		workflowLister:     workflowInformer.Lister(),
	}

	impl := workflowreconciler.NewImpl(ctx, reconciler)

	logger.Info("Setting up event handlers")

	workflowInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
