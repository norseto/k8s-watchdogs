# Project Overview

This repository hosts a collection of watchdog tools for Kubernetes clusters. The main CLI tool is `watchdogs`, which provides several subcommands.

## Directory Layout

- `cmd/watchdogs` - CLI entry point implementation.
- `internal/cmd` - Subcommands such as `clean-evicted`.
- `internal/options` - Flags and common options.
- `internal/rebalancer` - Pod relocation logic.
- `pkg/kube` - Kubernetes client helpers.
- `pkg/logger` - Logging utilities.
- `config` - Example manifests for CronJobs and other components.

## Development Tips

- Run `make vet` and `go test ./...` before committing changes.
- Format code with `go fmt ./...`.
- Write commit messages in English.

See `AGENTS.md` for more contribution details and rules.
