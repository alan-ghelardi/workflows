package workflow

import (
	"context"

	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"

	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"

	"github.com/nubank/workflows/pkg/apis/config"
	workflowsclient "github.com/nubank/workflows/pkg/client/injection/client"
	workflowinformer "github.com/nubank/workflows/pkg/client/injection/informers/workflows/v1alpha1/workflow"
	workflowreconciler "github.com/nubank/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	"github.com/nubank/workflows/pkg/github"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(ctx context.Context, watcher configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx)

	workflowInformer := workflowinformer.Get(ctx)

	configStore := config.NewStore(logger.Named("configs"))
	configStore.WatchConfigs(watcher)

	reconciler := &Reconciler{
		DeployKeys:         github.GetDeployKeyReconcilerOrDie(ctx),
		Webhook:            github.GetWebhookReconcilerOrDie(ctx),
		KubeClientSet:      kubeclient.Get(ctx),
		WorkflowsClientSet: workflowsclient.Get(ctx),
		workflowLister:     workflowInformer.Lister(),
	}

	impl := workflowreconciler.NewImpl(ctx, reconciler, func(*controller.Impl) controller.Options {
		return controller.Options{ConfigStore: configStore}
	})

	logger.Info("Setting up event handlers")

	workflowInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	return impl
}
