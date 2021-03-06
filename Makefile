IMAGE_REPOSITORY ?= 193814090748.dkr.ecr.us-east-1.amazonaws.com/workflows

# Run unit tests.
test:
	go test ./...

# Lint Go source files.
lint: golangci-lint
	@golangci-lint run

# Install golangci-lint if needed.
golangci-lint:
ifeq (, $(shell which golangci-lint))
	@{ \
		set -euo pipefail;\
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.31.0;\
	}
endif

# Install resources in the configured Kubernetes cluster in ~/.kube/config
install-dev: lint
	eval $$(minikube -p minikube docker-env) && kustomize  build config/base/ | ko apply -Lf -

install: lint
	KO_DOCKER_REPO=$(IMAGE_REPOSITORY) kustomize  build config/base/ | ko apply -Bf -
