package main

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/certificates"
	"knative.dev/pkg/webhook/configmaps"
	"knative.dev/pkg/webhook/resourcesemantics"
	"knative.dev/pkg/webhook/resourcesemantics/defaulting"
	"knative.dev/pkg/webhook/resourcesemantics/validation"

	"github.com/nubank/workflows/pkg/apis/config"
	workflowsv1alpha1 "github.com/nubank/workflows/pkg/apis/workflows/v1alpha1"
)

var types = map[schema.GroupVersionKind]resourcesemantics.GenericCRD{
	// List the types to validate.
	workflowsv1alpha1.SchemeGroupVersion.WithKind("Workflow"): &workflowsv1alpha1.Workflow{},
}

var callbacks = map[schema.GroupVersionKind]validation.Callback{}

func NewDefaultingAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	logger := logging.FromContext(ctx).Named("configs")
	configStore := config.NewStore(logger)
	configStore.WatchConfigs(cmw)

	return defaulting.NewAdmissionController(ctx,

		// Name of the resource webhook.
		fmt.Sprintf("defaulting.webhook.%s.workflows.dev", system.Namespace()),

		// The path on which to serve the webhook.
		"/defaulting",

		// The resources to default.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		configStore.WithConfig,

		// Whether to disallow unknown fields.
		true,
	)
}

func NewValidationAdmissionController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return validation.NewAdmissionController(ctx,

		// Name of the resource webhook.
		fmt.Sprintf("validation.webhook.%s.workflows.dev", system.Namespace()),

		// The path on which to serve the webhook.
		"/resource-validation",

		// The resources to validate.
		types,

		// A function that infuses the context passed to Validate/SetDefaults with custom metadata.
		func(ctx context.Context) context.Context {
			// Here is where you would infuse the context with state
			// (e.g. attach a store with configmap data)
			return ctx
		},

		// Whether to disallow unknown fields.
		true,

		// Extra validating callbacks to be applied to resources.
		callbacks,
	)
}

func NewConfigValidationController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	return configmaps.NewAdmissionController(ctx,

		// Name of the configmap webhook.
		fmt.Sprintf("config.webhook.%s.workflows.dev", system.Namespace()),

		// The path on which to serve the webhook.
		"/config-validation",

		// The configmaps to validate.
		configmap.Constructors{
			config.DefaultsConfigName: config.NewDefaultsFromConfigMap,
			logging.ConfigMapName():   logging.NewConfigFromConfigMap,
			metrics.ConfigMapName():   metrics.NewObservabilityConfigFromConfigMap,
		},
	)
}

func main() {
	ctx := webhook.WithOptions(signals.NewContext(), webhook.Options{
		ServiceName: "webhook",
		Port:        8443,
		SecretName:  "webhook-certs",
	})

	sharedmain.WebhookMainWithContext(ctx, "webhook",
		certificates.NewController,
		NewDefaultingAdmissionController,
		NewValidationAdmissionController,
		NewConfigValidationController,
	)
}
