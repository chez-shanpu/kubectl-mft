package registry

import (
	"context"
	"fmt"
	"os"
)

type Registry struct {
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Save(ctx context.Context, tag string, manifestPath string) error {
	ref, err := parseReference(tag)
	if err != nil {
		return err
	}

	fs, err := newFileStore(ctx, ref, manifestPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fs.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close manifestPath content: %v\n", closeErr)
		}
	}()

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return err
	}

	return copyRepo(ctx, fs, layoutStore, ref)
}

func (r *Registry) Push(ctx context.Context, tag string) error {
	ref, err := parseReference(tag)
	if err != nil {
		return err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return err
	}

	repo, err := newAuthenticatedRepository(ref)
	if err != nil {
		return err
	}

	return copyRepo(ctx, layoutStore, repo, ref)
}

func (r *Registry) Pull(ctx context.Context, tag string) error {
	ref, err := parseReference(tag)
	if err != nil {
		return err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return err
	}

	repo, err := newAuthenticatedRepository(ref)
	if err != nil {
		return err
	}

	return copyRepo(ctx, repo, layoutStore, ref)
}
