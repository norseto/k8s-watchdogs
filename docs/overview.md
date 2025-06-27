# Codebase Overview

This project provides a collection of CLI utilities to help operate Kubernetes clusters. It is written in Go and organized using standard Go modules.

## Directory Structure

- `cmd/watchdogs` - Entry point for the CLI binary.
- `internal/cmd` - Subcommands implementing each feature, such as pod clean up or rebalancing.
- `internal/options` - Common flag handling.
- `internal/rebalancer` - Logic for distributing pods evenly across nodes.
- `pkg/kube` - Helper functions for interacting with Kubernetes resources.
- `pkg/logger` - Simple wrapper around controller-runtime logging.
- `config/watchdogs` - Example manifests for running the tools as CronJobs.

## Building and Testing

Run `make build` to compile the binary. `go test ./...` runs the unit tests and `make vet` performs static analysis. The project uses `go fmt` for formatting which can be executed with `make fmt`.

## Release Information

The current release version is defined in `version.go`.
