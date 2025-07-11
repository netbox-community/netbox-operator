# Image URL to use all building/pushing image targets
IMG ?= your-image-registry:latest
LOCAL_IMG = netbox-operator:build-local #Should not be changed without changing kind/kustomization.yaml too
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.0

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

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GEN_DIR := gen/mock_interfaces
NETBOX_MOCKS_OUTPUT_FILE := netbox_mocks.go
INTERFACE_DEFITIONS_DIR := pkg/netbox/interfaces/netbox.go
GINKGO=ginkgo

GOFILES = $(shell find . -name \*.go ! -name 'zz_generated.*')

# check if govulncheck is installed or not
GO_PACKAGE_NAME_GOVULNCHECK := govulncheck
GO_PACKAGE_GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@latest
install-$(GO_PACKAGE_NAME_GOVULNCHECK):
	@if [ ! -x "$(GOBIN)/$(GO_PACKAGE_NAME_GOVULNCHECK)" ]; then \
		echo "Installing $(GO_PACKAGE_NAME_GOVULNCHECK)..." ; \
		go install $(GO_PACKAGE_GOVULNCHECK) ; \
	else \
		echo "$(GO_PACKAGE_NAME_GOVULNCHECK) is installed" ; \
	fi

# check if golangci-lint is installed or not
GO_PACKAGE_NAME_GOLANGCI_LINT := golangci-lint
install-$(GO_PACKAGE_NAME_GOLANGCI_LINT):
	@if [ ! -x "$(GOBIN)/$(GO_PACKAGE_NAME_GOLANGCI_LINT)" ]; then \
		echo "Installing $(GO_PACKAGE_NAME_GOLANGCI_LINT)..." ; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.60.3 ; \
	else \
		echo "$(GO_PACKAGE_NAME_GOLANGCI_LINT) is installed" ; \
	fi

# check if chainsaw is installed or not
GO_PACKAGE_NAME_CHAINSAW := chainsaw
install-$(GO_PACKAGE_NAME_CHAINSAW):
	@if [ ! -x "$(GOBIN)/$(GO_PACKAGE_NAME_CHAINSAW)" ]; then \
		echo "Installing $(GO_PACKAGE_NAME_CHAINSAW)..." ; \
		go install github.com/kyverno/chainsaw@v0.2.12 ; \
	else \
		echo "$(GO_PACKAGE_NAME_CHAINSAW) is installed" ; \
	fi

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
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt:
	@echo "Verifying gofmt"
	@!(gofmt -l -s -d ${GOFILES} | grep '[a-z]')

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: install-$(GO_PACKAGE_NAME_GOLANGCI_LINT) ## Run golangci-lint against code.
	golangci-lint run --config tools/.golangci.yaml ./...

.PHONY: vulncheck
vulncheck: install-$(GO_PACKAGE_NAME_GOVULNCHECK) ## Run govulncheck against code.
	govulncheck -show verbose ./...

.PHONY: test
test: manifests generate fmt vet ## Run tests.
	go test ./pkg/... -v 2>&1 -coverpkg=./... -tags='unit' -covermode=atomic -coverprofile=./unit_coverage.out

.PHONY: integration-test
integration-test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GINKGO) -cover -vv ./internal/controller/...
##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-build-local
docker-build-local: ## Build docker image with the manager.
	DOCKER_BUILDKIT=1 $(CONTAINER_TOOL) build -t ${LOCAL_IMG} -f Dockerfile .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

.PHONY: grafana
grafana: ## Generate grafana dashboard json files.
	kubebuilder edit --plugins grafana.kubebuilder.io/v1-alpha

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: create-kind
create-kind:
	./kind/local-env.sh

.PHONY: deploy-kind
deploy-kind: docker-build-local manifests kustomize
	kind load docker-image ${LOCAL_IMG}
	kind load docker-image ${LOCAL_IMG}  # fixes an issue with podman where the image is not correctly tagged after the first kind load docker-image
	$(KUSTOMIZE) build kind | $(KUBECTL) apply -f -

.PHONY: undeploy-kind
undeploy-kind: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build kind | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -
##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.16.4

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || GOBIN=$(LOCALBIN) GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@87bcfec

generate_mocks: ## TODO: auto install go install go.uber.org/mock/mockgen@latest
	mkdir -p ${GEN_DIR}
	mockgen -destination ${GEN_DIR}/${NETBOX_MOCKS_OUTPUT_FILE} -source=${INTERFACE_DEFITIONS_DIR}

# e2e tests
E2E_PARAM := --namespace e2e --parallel 3 --apply-timeout 3m --assert-timeout 3m --delete-timeout 3m --error-timeout 3m --exec-timeout 3m --cleanup-timeout 3m # --skip-delete (add this argument for local debugging)
.PHONY: create-kind-3.7.8
create-kind-3.7.8:
	./kind/local-env.sh --version 3.7.8
.PHONY: test-e2e-3.7.8
test-e2e-3.7.8: create-kind-3.7.8 deploy-kind install-$(GO_PACKAGE_NAME_CHAINSAW)
	chainsaw test $(E2E_PARAM)

.PHONY: create-kind-4.0.11
create-kind-4.0.11:
	./kind/local-env.sh --version 4.0.11
.PHONY: test-e2e-4.0.11
test-e2e-4.0.11: create-kind-4.0.11 deploy-kind install-$(GO_PACKAGE_NAME_CHAINSAW)
	chainsaw test $(E2E_PARAM)

.PHONY: create-kind-4.1.11
create-kind-4.1.11:
	./kind/local-env.sh --version 4.1.11
.PHONY: test-e2e-4.1.11
test-e2e-4.1.11: create-kind-4.1.11 deploy-kind install-$(GO_PACKAGE_NAME_CHAINSAW)
	chainsaw test $(E2E_PARAM)
