package mft

import "context"

type Registry interface {
	Save(ctx context.Context, tag string, manifestPath string) error
	Push(ctx context.Context, tag string) error
	Pull(ctx context.Context, tag string) error
}

// Pack packages a Kubernetes manifest into OCI layout format
func Pack(ctx context.Context, r Registry, tag string, manifest string) error {
	return r.Save(ctx, tag, manifest)
}

// Pull pulls a Kubernetes manifest from an OCI registry
func Pull(ctx context.Context, r Registry, tag string) error {
	return r.Pull(ctx, tag)
}

// Push pushes a Kubernetes manifest to an OCI registry
func Push(ctx context.Context, r Registry, tag string) error {
	return r.Push(ctx, tag)
}
