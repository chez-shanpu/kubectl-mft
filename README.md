# kubectl-mft

**The simplest way to manage Kubernetes manifests**

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)

kubectl-mft is a kubectl plugin that makes manifest management as simple as managing container images. No complex templating, no overlay structures—just save, version, and retrieve your manifests.

## Why kubectl-mft?

**Simpler than Helm**
- No chart structure or templating syntax to learn
- No values.yaml files to maintain
- Just plain Kubernetes YAML

**Simpler than Kustomize**
- No base/overlay directory structure
- No patches or strategic merge logic
- Store and retrieve manifests directly

**As simple as Docker images**

```bash
kubectl mft pack -f deployment.yaml -t myregistry/app:v1.0.0
kubectl mft push -t myregistry/app:v1.0.0
kubectl mft pull -t myregistry/app:v1.0.0
kubectl mft dump -t myregistry/app:v1.0.0 | kubectl apply -f -
```

Under the hood, it uses OCI registries—the same technology that stores your container images.

## Features

- **Simple workflow** - Pack, push, pull, and apply—just like Docker
- **Version control** - Tag and version your manifests like container images
- **Any OCI registry** - Works with Docker Hub, GitHub Container Registry, Google Artifact Registry, etc.
- **Local caching** - Efficiently manage locally stored manifests

## Quick Start

```bash
# Package a Kubernetes manifest
kubectl mft pack -f deployment.yaml -t localhost:5000/myapp/config:v1.0.0

# Push to OCI registry
kubectl mft push -t localhost:5000/myapp/config:v1.0.0

# Pull from OCI registry
kubectl mft pull -t localhost:5000/myapp/config:v1.0.0

# Dump and apply to cluster
kubectl mft dump -t localhost:5000/myapp/config:v1.0.0 | kubectl apply -f -
```

## Installation

### Using Go

```bash
go install github.com/chez-shanpu/kubectl-mft@latest
```

### From Source

```bash
git clone https://github.com/chez-shanpu/kubectl-mft.git
cd kubectl-mft
make build
# Binary will be in bin/kubectl-mft
```

## Usage Examples

### Basic Workflow

1. **Pack a manifest into OCI layout**

```bash
kubectl mft pack -f my-deployment.yaml -t ghcr.io/myorg/manifests:v1.0.0
```

2. **Push to a registry**

```bash
# Authenticate first (if needed)
docker login ghcr.io

# Push the manifest
kubectl mft push -t ghcr.io/myorg/manifests:v1.0.0
```

3. **Pull from a registry**

```bash
kubectl mft pull -t ghcr.io/myorg/manifests:v1.0.0
```

4. **Apply to cluster**

```bash
kubectl mft dump -t ghcr.io/myorg/manifests:v1.0.0 | kubectl apply -f -
```

### Managing Local Manifests

**List all locally stored manifests**

```bash
# Table format (default)
kubectl mft list

# JSON format
kubectl mft list -o json

# YAML format
kubectl mft list -o yaml
```

**Get file path to manifest blob**

```bash
kubectl mft path -t localhost:5000/myapp:v1.0.0
```

**Delete a manifest**

```bash
# With confirmation prompt
kubectl mft delete -t localhost:5000/myapp:v1.0.0

# Skip confirmation
kubectl mft delete -t localhost:5000/myapp:v1.0.0 --force
```

**Save manifest to file**

```bash
kubectl mft dump -t ghcr.io/myorg/manifests:v1.0.0 -o my-manifest.yaml
```

## Command Reference

| Command | Description |
|---------|-------------|
| `pack` | Package a Kubernetes manifest into OCI layout format |
| `push` | Push a manifest to an OCI registry |
| `pull` | Pull a manifest from an OCI registry |
| `dump` | Output a manifest from local storage |
| `list` | List all locally stored manifests |
| `path` | Get the file path to a manifest blob |
| `delete` | Delete a manifest from local storage |

For detailed usage of each command, run `kubectl mft <command> --help`.

## Authentication

kubectl-mft uses Docker's credential store for registry authentication. Log in using Docker:

```bash
docker login registry.example.com
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

Copyright Authors of kubectl-mft
