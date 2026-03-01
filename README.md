<img src="image.jpg" alt="kubectl-mft" width="100%">

<!-- Image generated with Gemini Nano Banana Pro -->

# kubectl-mft

**The simplest way to manage Kubernetes manifests**

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev/)

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
kubectl mft pack -f deployment.yaml myregistry/app:v1.0.0
kubectl mft push myregistry/app:v1.0.0
kubectl mft apply myregistry/app:v1.0.0
```

Under the hood, it uses OCI registries—the same technology that stores your container images.

## Features

- **Simple workflow** - Pack, push, pull, and apply—just like Docker
- **Version control** - Tag and version your manifests like container images
- **Any OCI registry** - Works with Docker Hub, GitHub Container Registry, Google Artifact Registry, etc.
- **Manifest validation** - Validate Kubernetes manifests against schemas before packing, with CRD support
- **Manifest signing** - Sign and verify manifests with ECDSA P-256 keys
- **Local caching** - Efficiently manage locally stored manifests

## Quick Start

```bash
# Package a Kubernetes manifest
kubectl mft pack -f deployment.yaml localhost:5000/myapp/config:v1.0.0

# Push to OCI registry
kubectl mft push localhost:5000/myapp/config:v1.0.0

# Pull from OCI registry
kubectl mft pull localhost:5000/myapp/config:v1.0.0

# Apply to cluster (auto-pulls if not local)
kubectl mft apply localhost:5000/myapp/config:v1.0.0
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
kubectl mft pack -f my-deployment.yaml ghcr.io/myorg/manifests:v1.0.0
```

2. **Push to a registry**

```bash
# Authenticate first (if needed)
docker login ghcr.io

# Push the manifest
kubectl mft push ghcr.io/myorg/manifests:v1.0.0
```

3. **Pull from a registry**

```bash
kubectl mft pull ghcr.io/myorg/manifests:v1.0.0
```

4. **Apply to cluster**

```bash
# Auto-pulls from the registry if not already stored locally
kubectl mft apply ghcr.io/myorg/manifests:v1.0.0
```

### Simple Tag Names

You can use simple tag names without a registry prefix. They are automatically stored under the `local/` namespace:

```bash
# These are equivalent:
kubectl mft pack -f deployment.yaml myapp:v1.0.0
kubectl mft pack -f deployment.yaml local/myapp:v1.0.0

# List shows them without the "local/" prefix
kubectl mft list
# REPOSITORY   TAG      SIZE   CREATED
# myapp        v1.0.0   694B   2025-01-15 10:30

# Apply using simple tag
kubectl mft apply myapp:v1.0.0
```

### Signing and Verification

kubectl-mft supports signing manifests with ECDSA P-256 keys. Signing happens automatically during `pack`, and verification during `pull`.

**Initial setup (one-time)**

```bash
# Generate a signing key pair
kubectl mft key generate

# Share your public key with verifiers
kubectl mft key export > my-public-key.pub
```

**Signer workflow**

```bash
# Pack automatically signs the manifest
kubectl mft pack -f deployment.yaml myregistry/app:v1.0.0
kubectl mft push myregistry/app:v1.0.0

# Skip signing if no key is available
kubectl mft pack -f deployment.yaml --skip-sign myregistry/app:v1.0.0
```

**Verifier workflow**

```bash
# Import a public key for verification
kubectl mft key import signer-public-key.pub --name alice

# Pull automatically verifies the signature
kubectl mft pull myregistry/app:v1.0.0

# Skip verification if no key is available
kubectl mft pull --skip-verify myregistry/app:v1.0.0

# Apply without signature verification
kubectl mft apply --skip-verify myregistry/app:v1.0.0
```

**Standalone sign and verify**

```bash
kubectl mft sign myregistry/app:v1.0.0
kubectl mft verify myregistry/app:v1.0.0
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
kubectl mft path localhost:5000/myapp:v1.0.0

# Use with kubectl debug --custom
kubectl debug mypod -it --image busyboz --custom=$(kubectl mft path localhost:5000/debug-container)
```

> **Note:** When packing YAML files that do not contain `apiVersion`/`kind` fields (e.g., debug container custom profiles), a warning like the following will be printed to stderr, but the pack operation completes successfully and the file is stored correctly.
> ```
> warning: debug-profile.yaml: error while parsing: missing 'kind' key
> ```
> To suppress this warning, use the `--skip-validation` flag: `kubectl mft pack --skip-validation ...`

**Delete a manifest**

```bash
# With confirmation prompt
kubectl mft delete localhost:5000/myapp:v1.0.0

# Skip confirmation
kubectl mft delete localhost:5000/myapp:v1.0.0 --force
```

**Save manifest to file**

```bash
kubectl mft dump ghcr.io/myorg/manifests:v1.0.0 -o my-manifest.yaml
```

**Copy a manifest to a new tag**

```bash
# Copy within the same repository
kubectl mft cp ghcr.io/myorg/manifests:v1.0.0 ghcr.io/myorg/manifests:latest

# Copy to a different repository
kubectl mft cp ghcr.io/myorg/manifests:v1.0.0 ghcr.io/myorg/prod-manifests:v1.0.0
```

### Manifest Validation

kubectl-mft validates your Kubernetes manifests when packing to catch errors early.

**Basic validation (automatic)**

```bash
# Validation runs automatically during pack
kubectl mft pack -f deployment.yaml myapp:v1.0.0

# Skip validation if needed
kubectl mft pack -f deployment.yaml myapp:v1.0.0 --skip-validation
```

**Register CRD schemas for custom resource validation**

```bash
# Register a CRD schema
kubectl mft schema add -f myresource-crd.yaml

# List registered schemas
kubectl mft schema list

# Delete a registered schema
kubectl mft schema delete example.com/MyResource
```

**Multi-document YAML support**

Manifests with multiple resources separated by `---` are validated individually:

```bash
# Each resource in the file is validated separately
kubectl mft pack -f multi-resource.yaml myapp:v1.0.0
```

## Command Reference

| Command | Description |
|---------|-------------|
| `pack` | Package and validate a Kubernetes manifest into OCI layout format |
| `push` | Push a manifest to an OCI registry |
| `pull` | Pull a manifest from an OCI registry |
| `apply` | Apply a manifest to the current Kubernetes cluster (auto-pulls if not local) |
| `dump` | Output a manifest from local storage |
| `list` | List all locally stored manifests |
| `path` | Get the file path to a manifest blob |
| `delete` | Delete a manifest from local storage |
| `cp` | Copy a manifest to a new tag in local storage |
| `sign` | Sign a packed manifest |
| `verify` | Verify the signature of a manifest |
| `key generate` | Generate an ECDSA P-256 key pair for signing |
| `key import` | Import a public key for signature verification |
| `key export` | Export a public key to stdout |
| `key list` | List all signing keys |
| `key delete` | Delete a public key |
| `schema add` | Register a CRD schema for custom resource validation |
| `schema list` | List registered CRD schemas |
| `schema delete` | Delete a registered CRD schema |

For detailed usage of each command, run `kubectl mft <command> --help`.

## Authentication

kubectl-mft uses Docker's credential store for registry authentication. Log in using Docker:

```bash
docker login registry.example.com
```

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

Copyright Authors of kubectl-mft
