<img src="image.jpg" alt="kubectl-mft" width="100%">

<!-- Image generated with Gemini Nano Banana Pro -->

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

### Using Krew (Recommended)

[Krew](https://krew.sigs.k8s.io/) is the plugin manager for kubectl.

```bash
# Add the custom index
kubectl krew index add mft https://github.com/chez-shanpu/kubectl-mft.git

# Install kubectl-mft
kubectl krew install mft/mft

# Verify installation
kubectl mft --help
```

To update:

```bash
kubectl krew upgrade mft/mft
```

### Download Binary from GitHub Releases

Download the latest release for your platform from [GitHub Releases](https://github.com/chez-shanpu/kubectl-mft/releases).

**Linux / macOS**

```bash
# Download and extract (replace VERSION, OS, and ARCH as needed)
curl -L https://github.com/chez-shanpu/kubectl-mft/releases/download/VERSION/kubectl-mft_VERSION_OS_ARCH.tar.gz | tar xz

# Move to a directory in your PATH
sudo mv kubectl-mft /usr/local/bin/

# Verify installation
kubectl mft --help
```

**Windows**

Download the `.zip` file for your architecture from the releases page, extract it, and add the binary to your PATH.

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

### Simple Tag Names

You can use simple tag names without a registry prefix. They are automatically stored under the `local/` namespace:

```bash
# These are equivalent:
kubectl mft pack -f deployment.yaml -t myapp:v1.0.0
kubectl mft pack -f deployment.yaml -t local/myapp:v1.0.0

# List shows them without the "local/" prefix
kubectl mft list
# REPOSITORY   TAG      SIZE   CREATED
# myapp        v1.0.0   694B   2025-01-15 10:30

# Dump using simple tag
kubectl mft dump -t myapp:v1.0.0 | kubectl apply -f -
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
# Get the file path
kubectl mft path -t localhost:5000/myapp:v1.0.0

# Use with kubectl debug --custom
kubectl debug mypod -it --image busyboz --custom=$(kubectl mft path -t localhost:5000/debug-container)
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

**Copy a manifest to a new tag**

```bash
# Copy within the same repository
kubectl mft cp ghcr.io/myorg/manifests:v1.0.0 ghcr.io/myorg/manifests:latest

# Copy to a different repository
kubectl mft cp ghcr.io/myorg/manifests:v1.0.0 ghcr.io/myorg/prod-manifests:v1.0.0
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
| `cp` | Copy a manifest to a new tag in local storage |

For detailed usage of each command, run `kubectl mft <command> --help`.

## Authentication

kubectl-mft uses Docker's credential store for registry authentication. Log in using Docker:

```bash
docker login registry.example.com
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

Copyright Authors of kubectl-mft
