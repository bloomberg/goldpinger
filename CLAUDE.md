# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Goldpinger is a networking monitoring tool for Kubernetes clusters that runs as a DaemonSet and monitors connectivity between pods. It generates network topology visualizations, Prometheus metrics, and provides an API for cluster health monitoring.

## Development Commands

### Build Commands

- `make bin/goldpinger` - Compile the binary locally
- `make build` - Build Docker image using multi-stage Dockerfile
- `make build-release` - Build and push multi-arch images (linux/amd64, linux/arm64)
- `make run` - Run the application directly with `go run`

### Testing and Quality

- `go test ./...` - Run all tests
- `go vet ./...` - Run Go vet for static analysis
- `go fmt ./...` - Format Go code

### Development Tools

- `make swagger` - Regenerate API client/server code from swagger.yml
- `make vendor` - Download and vendor dependencies
- `make clean` - Clean build artifacts

## Architecture

### Core Components

**Main Application (`cmd/goldpinger/main.go`)**

- Entry point that initializes logging, Kubernetes client, and starts the REST API server
- Handles configuration parsing and command-line flags
- Sets up structured logging with Zap

**Core Logic (`pkg/goldpinger/`)**

- `config.go` - Configuration management and command-line flags
- `k8s.go` - Kubernetes API integration and pod discovery
- `pinger.go` - Network connectivity testing logic
- `client.go` - HTTP client for inter-pod communication
- `updater.go` - Background processes for continuous monitoring
- `stats.go` - Metrics collection and Prometheus integration
- `probes.go` - Health check and probe implementations

**REST API (`pkg/restapi/`)**

- Auto-generated from `swagger.yml` using go-swagger
- Provides endpoints for health checks, ping operations, and cluster status
- Never edit generated files directly - modify `swagger.yml` and run `make swagger`

**React SPA Frontend (`web/`)**

- Modern React-based SPA with TypeScript and TanStack Start framework
- Uses Vite for bundling and TanStack Router for routing
- Assets are embedded directly in the Go binary for single-artifact deployment
- Supports development mode with hot reloading via proxy to Vite dev server

### Key Features

**Network Monitoring**

- Ping connectivity between all pods in the cluster
- DNS resolution testing for configured hostnames
- TCP and HTTP checks to external targets
- IPv4/IPv6 dual-stack support

**Metrics and Observability**

- Prometheus metrics at `/metrics` endpoint
- Grafana dashboard templates in `extras/`
- Structured logging with configurable Zap logger

**Kubernetes Integration**

- Uses in-cluster config or kubeconfig for API access
- Discovers pods via label selectors (`app=goldpinger`)
- Supports namespace filtering and RBAC

## Configuration

**Environment Variables**

- `HOST` - Bind address (default: 0.0.0.0)
- `PORT` - Server port (default: 8080)
- `HOSTNAME` - Node hostname for metrics
- `POD_IP` - Pod IP for peer selection
- `HOSTS_TO_RESOLVE` - Space-delimited hostnames for DNS testing
- `TCP_TARGETS` - External TCP endpoints to check
- `HTTP_TARGETS` - External HTTP endpoints to check

**Configuration Files**

- `config/zap.json` - Structured logging configuration
- `swagger.yml` - API specification (regenerate code after changes)
- `web/vite.config.ts` - React build configuration
- `web/package.json` - React dependencies and scripts

## Frontend Development

**Development Workflow**

1. Start the Go backend: `make run --dev-mode` (enables proxy to Vite)
2. Start the React dev server: `cd web && npm run dev`
3. Access the application at `http://localhost:8080` (proxies to Vite on port 3000)

**Production Build**

- React assets are embedded in the Go binary using `go:embed`
- Single binary deployment with no external static file dependencies
- Build process: `make build-web` → `make bin/goldpinger`

## Deployment

Goldpinger is designed to run as a Kubernetes DaemonSet with appropriate RBAC permissions. See the Helm chart in `charts/goldpinger/` for production deployment configuration.

The application requires cluster-level permissions to list pods across namespaces for connectivity testing. The React frontend is served directly from the embedded assets in the Go binary.
