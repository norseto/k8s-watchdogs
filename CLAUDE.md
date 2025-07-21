# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Testing
```bash
# Format code
make fmt

# Run static analysis
make vet

# Run all tests with coverage
make test

# Build binary
make build

# Run CLI directly
make run

# Run single test file
go test ./internal/cmd/clean-evicted -v

# Run specific test
go test ./internal/cmd/clean-evicted -run TestCleanEvicted -v
```

### Code Quality and Security
```bash
# Run golangci-lint
make lint

# Run golangci-lint with automatic fixes
make lint-fix

# Run vulnerability check with govulncheck
make vulcheck

# Run security analysis with gosec
make seccheck
```

### Docker Operations
```bash
# Build Docker image
make docker-build

# Multi-arch build and push
make docker-buildx
```

### RBAC Generation
```bash
# Generate RBAC roles from code annotations
make roles
```

## Architecture Overview

**k8s-watchdogs** is a Kubernetes maintenance CLI tool that runs as CronJobs to perform automated cluster housekeeping tasks.

### Core Architecture Pattern

Each watchdog command follows a consistent pattern:
1. **Command Definition**: Located in `internal/cmd/{command-name}/` with Cobra CLI setup
2. **Kubernetes Client**: Uses shared client utilities from `pkg/kube/client/` 
3. **Resource Operations**: Leverages resource-specific utilities in `pkg/kube/` (pod.go, deployment.go, etc.)
4. **Logging**: Structured logging via `pkg/logger/` using logr interface

### Key Components

- **Main Entry**: `cmd/watchdogs/main.go` sets up Cobra CLI with all subcommands
- **Client Setup**: `pkg/kube/client/clientutils.go` handles kubeconfig loading and client creation
- **Resource Utilities**: `pkg/kube/{resource}.go` files provide typed operations for K8s resources
- **Rebalancer**: `internal/rebalancer/` contains pod distribution logic for the rebalance-pods command

### Command Structure

All commands implement the same pattern:
- Accept namespace and resource-specific flags
- Use shared kubernetes client from context
- Log operations with structured logging
- Handle errors gracefully with appropriate exit codes

### Deployment Model

- Runs as Kubernetes CronJob (default: every minute)
- Uses ServiceAccount with minimal RBAC permissions
- Distroless container image for security
- Configurable via `config/watchdogs/` manifests

### Testing Strategy

- Unit tests alongside implementation files (`*_test.go`)
- Kubernetes client operations are typically mocked
- Coverage reports generated to `cover.out`
- Test individual commands: `go test ./internal/cmd/{command}/`