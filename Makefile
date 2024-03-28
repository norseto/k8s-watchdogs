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
roles:
	controller-gen rbac:roleName=k8s-watchdogs-role paths=./... output:dir=config/watchdogs/rbac/
