# Codebase Overview

This document provides a brief overview of the repository to help new contributors navigate the codebase.

## Directory Layout

- `cmd/` – entry point for the command line interface.
- `internal/` – implementation of CLI commands and shared utilities.
  - `cmd/` – individual subcommands such as `clean-evicted`.
  - `options/` – common flag bindings.
  - `rebalancer/` – logic used by the `rebalance-pods` command.
- `pkg/` – reusable packages.
  - `kube/` – helper functions for interacting with Kubernetes resources.
  - `logger/` – logger setup and helpers.
  - `generics/` – small generic utilities.
- `config/watchdogs/` – sample Kubernetes manifests for running the tool as a CronJob.
- `hack/` – helper scripts and misc resources used during development.

## Building and Testing

Use the `Makefile` for common tasks:

- `make fmt` – run `go fmt` on all packages.
- `make vet` – run `go vet`.
- `make test` – run the unit tests.
- `make build` – compile the binary to `bin/k8s-watchdogs`.

Docker images can be built using `make docker-build` or `make docker-buildx`.

## Versioning

Version information is stored in `version.go`. The Git commit hash is injected at build time using the `GITVERSION` build argument in the Makefile.

## Contribution Notes

- Run `make fmt` and `make vet` before committing changes.
- Ensure `go test ./...` passes locally.
- Commit messages should follow Conventional Commits.

