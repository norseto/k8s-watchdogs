# Codebase Overview

This document provides a high-level overview of the repository for new contributors.

## Directory Structure

- `cmd/` - Entry point for the `watchdogs` CLI application.
- `internal/` - Application commands and helper packages used only within this project.
  - `cmd/` - Individual subcommands such as `clean-evicted` and `rebalance-pods`.
  - `options/` - Common command-line option parsing utilities.
  - `rebalancer/` - Logic for pod rebalancing.
- `pkg/` - Public packages that can be reused by other modules.
  - `kube/` - Kubernetes helpers for interacting with pods, nodes, deployments and statefulsets.
  - `logger/` - Simple logging initialization.
  - `generics/` - Generic utility functions.
- `config/` - Example Kubernetes manifests for running the cron jobs.
- `hack/` - Helper scripts for release and version management.

## Build and Test

- Format code with `go fmt ./...` or `make fmt`.
- Run `make vet` for linting and `go test ./...` before submitting changes.
- Use the `Makefile` to build (`make build`) or run (`make run`) the CLI.

## Versioning

`version.go` defines the current release version and is updated during releases.

## Contribution

Please follow conventional commit messages and keep commit and branch names in English. Tests should pass before opening a pull request.
