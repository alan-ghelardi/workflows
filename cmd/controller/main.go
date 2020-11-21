package main

import (
	"context"

	corev1 "k8s.io/api/core/v1"

	"github.com/nubank/workflows/pkg/github"
	"github.com/nubank/workflows/pkg/reconciler/workflow"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
)

func main() {
	githubClient := github.NewClient()

	ctx := injection.WithNamespaceScope(context.Background(), corev1.NamespaceAll)

	ctx = github.WithDeployKeySyncer(ctx, githubClient)
	ctx = github.WithWebhookSyncer(ctx, githubClient)

	sharedmain.MainWithContext(ctx, "workflows-controller", workflow.NewController)
}
