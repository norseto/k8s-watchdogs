# Image URL to use all building/pushing image targets
IMG ?= k8s-watchdogs:latest
MODULE_PACKAGE=github.com/norseto/k8s-watchdogs
GITSHA := $(shell git describe --always)
LDFLAGS := -ldflags=all=

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: modernize
modernize: ## Run modernize against code. see https://pkg.go.dev/golang.org/x/tools/gopls/internal/analysis/modernize
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...

.PHONY: vulcheck
vulcheck: govulncheck ## Run govulncheck against code.
	$(GOVULNCHECK) ./...

.PHONY: seccheck
seccheck: gosec ## Run gosec against code.
	$(GOSEC) ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint with --fix to automatically fix issues.
	$(GOLANGCI_LINT) run --fix

.PHONY: roles
roles: controller-gen
	$(LOCALBIN)/controller-gen rbac:roleName=k8s-watchdogs-role paths=./... output:dir=config/watchdogs

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOVULNCHECK ?= $(LOCALBIN)/govulncheck
GOSEC ?= $(LOCALBIN)/gosec
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.16.5
GOVULNCHECK_VERSION ?= latest
GOSEC_VERSION ?= latest
GOLANGCI_LINT_VERSION ?= v2.4.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	# Run in module root so toolchain from go.mod is honored.
	GOBIN=$(LOCALBIN) go -C . install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK) ## Download govulncheck locally if necessary.
$(GOVULNCHECK): $(LOCALBIN)
	# Run in module root so toolchain from go.mod is honored.
	GOBIN=$(LOCALBIN) go -C . install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: gosec
gosec: $(GOSEC) ## Download gosec locally if necessary.
$(GOSEC): $(LOCALBIN)
	# Run in module root so toolchain from go.mod is honored.
	GOBIN=$(LOCALBIN) go -C . install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	# Run in module root so toolchain from go.mod is honored.
	GOBIN=$(LOCALBIN) go -C . install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: docker-buildx-setup
docker-buildx-setup: ## Setup buildx builder for multi-arch builds.
	docker buildx create --use --name multiarch-builder || true
.PHONY: docker-buildx
docker-buildx: docker-buildx-setup ## Build and push docker image for multiple architectures using buildx.
	docker buildx build  --build-arg MODULE_PACKAGE=$(MODULE_PACKAGE) --build-arg GITVERSION=$(GITSHA) --platform linux/amd64,linux/arm64 --push -t $(IMG) .
.PHONY: docker-buildx-local
docker-buildx-local: docker-buildx-setup ## Build multi-arch image locally (no push), output as OCI archive tar.
	mkdir -p dist
	docker buildx build --build-arg MODULE_PACKAGE=$(MODULE_PACKAGE) --build-arg GITVERSION=$(GITSHA) --platform linux/amd64,linux/arm64 -t $(IMG) --output=type=oci,dest=dist/k8s-watchdogs_oci.tar .
.PHONY: docker-build
docker-build:
	docker build  --build-arg MODULE_PACKAGE=$(MODULE_PACKAGE) --build-arg GITVERSION=$(GITSHA) -t $(IMG) .

.PHONY: build
build: vet ## Build manager binary.
	go build $(LDFLAGS)"-X $(MODULE_PACKAGE).GitVersion=$(GITSHA)" -o bin/k8s-watchdogs cmd/watchdogs/main.go

.PHONY: run
run: vet ## Run a controller from your host.
	go run $(LDFLAGS)"-X $(MODULE_PACKAGE).GitVersion=$(GITSHA)" ./cmd/watchdogs/main.go
