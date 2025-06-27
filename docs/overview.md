# Codebase Overview

This document introduces the main components of the **k8s-watchdogs** repository.

## Project Layout

- **cmd/**: entry point for the command line interface. `watchdogs` is the main executable.
- **internal/**: internal packages that contain subcommands, shared options and logic such as the rebalancer.
  - `cmd/` holds implementations of each command.
  - `options/` defines CLI options shared across commands.
  - `rebalancer/` contains logic for balancing pods across nodes.
- **pkg/**: reusable libraries.
  - `generics/` provides helper utilities.
  - `kube/` wraps common Kubernetes operations.
  - `logger/` configures structured logging.
- **config/**: Kubernetes manifests including the sample CronJob for running watchdogs.
- **hack/**: helper scripts for version management and releases.

## Building and Testing

Use `make` targets defined in the `Makefile`:

```bash
make fmt   # format the code
make vet   # run static analysis
make test  # execute all unit tests
```

The binary can be built with `make build`, while `make run` runs the CLI directly.

## Running in a Cluster

The `config/watchdogs` directory contains a CronJob manifest. Apply it after building and pushing a container image to your registry. Adjust the schedule and namespace as required.

## Contribution Tips

- Keep code formatted using `go fmt` or `make fmt`.
- Run `make vet` and `go test ./...` before committing changes.
- Commit messages follow the Conventional Commits style.

