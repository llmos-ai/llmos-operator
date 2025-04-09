# Common settings
REGISTRY ?= ghcr.io/llmos-ai
IMG_REPO ?= ${REGISTRY}/llmos-operator
WEBHOOK_IMG_REPO ?= ${REGISTRY}/llmos-operator-webhook

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.30.3

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
GLOBALBIN ?= /usr/local/bin

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize-$(KUSTOMIZE_VERSION)
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen-$(CONTROLLER_TOOLS_VERSION)
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
HELM ?= $(GLOBALBIN)/helm

## Tool Versions
KUSTOMIZE_VERSION ?= v5.3.0
CONTROLLER_TOOLS_VERSION ?= v0.14.0
ENVTEST_VERSION ?= release-0.17
GOLANGCI_LINT_VERSION ?= v1.62.2

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
export CONTROLLER_GEN
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	bash ./hack/generate-manifest

.PHONY: generate
export CONTROLLER_GEN
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	go generate

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter & yamllint
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
.PHONY: test-e2e  # Run the e2e tests against a Kind k8s instance that is spun up.
test-e2e:
	#go test ./test/e2e/ -v -ginkgo.v
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

##@ Build
.PHONY: build
build: lint test build-operator build-system-charts build-installer build-upgrade ## Run all llmos-operator builds

.PHONY: release
release: lint test release-operator package-system-charts-repo package-installer package-upgrade ## Run all llmos-operator builds

.PHONY: release-operator
release-operator: ## release llmos-operator using goreleaser.
	EXPORT_ENV=true source ./scripts/version && \
	goreleaser release --clean

.PHONY: build-operator
build-operator: ## Build llmos-operator using goreleaser with local mode.
	EXPORT_ENV=true source ./scripts/version && \
	goreleaser release --snapshot --clean $(VERBOSE)

.PHONY: push-manifests
push-manifests: operator-manifest webhook-manifest ## Push llmos-operator and webhook manifest image
	@echo "Build & pushed llmos-operator && llmos-operator-webhook manifest"

.PHONY: build-installer
build-installer: ## Build installer artifacts (i.e., operator charts & index.yaml)
	@echo Building llmos system-installer assets
	EXPORT_ENV=true source ./scripts/version && \
	earthly +build-installer

.PHONY: package-installer
package-installer: ## Build installer image using earthly (multi-arch).
	@echo Packaging llmos-operator installer image
	EXPORT_ENV=true source ./scripts/version && \
	earthly --push +package-all-installer

.PHONY: package-installer-local
package-installer-local: ## Build local llmos-operator to dist/charts(local arch).
	@echo Packaging llmos system-installer image
	EXPORT_ENV=true source ./scripts/version && \
	earthly -i +package-installer

.PHONY: build-system-charts
build-system-charts: ## build LLMOS system-charts-repo image
	@echo Packaging llmos system-charts repo
	EXPORT_ENV=true source ./scripts/version && \
	earthly +build-system-charts

.PHONY: package-system-charts-repo
package-system-charts-repo: ## Package LLMOS system-charts-repo image
	@echo Packaging llmos system-charts repo
	EXPORT_ENV=true source ./scripts/version && \
	earthly --push +package-all-system-charts-repo

.PHONY: build-upgrade
build-upgrade: ## Build upgrade image
	@echo Building upgrade image
	EXPORT_ENV=true source ./scripts/version && \
	earthly +build-upgrade

.PHONY: package-upgrade
package-upgrade: ## Package multi-arch upgrade manifest
	@echo Building upgrade image
	EXPORT_ENV=true source ./scripts/version && \
	earthly --push +package-all-upgrade-image

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: operator-manifest
operator-manifest: ## Build & push operator manifest image
	./scripts/manifest-images llmos-operator

.PHONY: webhook-manifest
webhook-manifest: ## Build & push webhook manifest image
	./scripts/manifest-images llmos-operator-webhook

.PHONY: ci
ci: ## Run ci script
	bash ./scripts/ci

##@ Chart
.PHONY: install
install: ## Install llmos-operator chart into the K8s cluster.
	$(HELM) upgrade --install --create-namespace -n llmos-system llmos-operator deploy/charts/llmos-operator \
	--reuse-values -f deploy/charts/llmos-operator/values.yaml

.PHONY: uninstall
uninstall: ## Uninstall llmos-operator chart from the K8s cluster.
	$(HELM) uninstall -n llmos-system llmos-operator

.PHONY: install-crds
install-crds: manifests ## Install CRDs into your k8s cluster.
	$(HELM) upgrade --install --create-namespace -n llmos-system llmos-crd deploy/charts/llmos-crd

.PHONY: uninstall-crds
uninstall-crds: ## Uninstall CRDs from your k8s cluster.
	$(HELM) uninstall -n llmos-system llmos-crd

.PHONY: helm-dep
helm-dep: ## update operator dependency charts.
	$(HELM) dep update deploy/charts/llmos-operator

.PHONY: helm-dep-system-charts
helm-dep-system-charts: ## update system-charts dependencies.
	$(HELM) dep update deploy/charts/system-charts

ifndef ignore-not-found
  ignore-not-found = false
endif

##@ Dependencies
## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef
