# Codebase Overview

This document gives new contributors a quick orientation to the repository.

## Directory Layout

- `cmd/` contains the main entry point. `watchdogs/main.go` wires commands and initializes logging.
- `internal/cmd/` holds subcommands for the CLI. Each command focuses on a specific Kubernetes operation such as cleaning evicted pods or restarting deployments.
- `internal/options/` provides helper functions for common CLI flags.
- `internal/rebalancer/` implements pod rebalancing logic across cluster nodes.
- `pkg/` packages reusable utilities:
  - `kube/` wraps Kubernetes client logic.
  - `generics/` defines generic helper functions.
  - `logger/` configures structured logging.
- `config/` contains sample Kubernetes manifests for running the tool via CronJob.
- `hack/` keeps scripts and helpers used during development.

## Building

Run `make build` to compile the binary. The Makefile also includes `make test`, `make fmt`, and `make vet` targets.

## Testing

Execute `go test ./...` to run all unit tests. Linting is done with `go vet`.

## Contribution Tips

1. Format code using `go fmt ./...` or `make fmt`.
2. Run `make vet` and `go test ./...` before committing.
3. Follow Conventional Commits for commit messages.

