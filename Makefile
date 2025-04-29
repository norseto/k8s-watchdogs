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

.PHONY: roles
roles: controller-gen
	$(LOCALBIN)/controller-gen rbac:roleName=k8s-watchdogs-role paths=./... output:dir=config/watchdogs

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.16.5

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: docker-buildx-setup
docker-buildx-setup: ## Setup buildx builder for multi-arch builds.
	docker buildx create --use --name multiarch-builder || true
.PHONY: docker-buildx
docker-buildx: docker-buildx-setup ## Build and push docker image for multiple architectures using buildx.
	docker buildx build  --build-arg MODULE_PACKAGE=$(MODULE_PACKAGE) --build-arg GITVERSION=$(GITSHA) --platform linux/amd64,linux/arm64 --push -t $(IMG) .
.PHONY: docker-build
docker-build:
	docker build  --build-arg MODULE_PACKAGE=$(MODULE_PACKAGE) --build-arg GITVERSION=$(GITSHA) -t $(IMG) .

.PHONY: build
build: vet ## Build manager binary.
	go build $(LDFLAGS)"-X $(MODULE_PACKAGE).GitVersion=$(GITSHA)" -o bin/k8s-watchdogs cmd/watchdogs/main.go

.PHONY: run
run: vet ## Run a controller from your host.
	go run $(LDFLAGS)"-X $(MODULE_PACKAGE).GitVersion=$(GITSHA)" ./cmd/watchdogs/main.go
