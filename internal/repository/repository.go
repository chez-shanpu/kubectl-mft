// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

type Repository struct {
	tag string
}

func NewRepository(tag string) *Repository {
	return &Repository{tag: tag}
}

func (r *Repository) Dump(ctx context.Context) ([]byte, error) {
	ref, err := parseReference(r.tag)
	if err != nil {
		return nil, err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return nil, err
	}

	desc, err := layoutStore.Resolve(ctx, ref.ReferenceOrDefault())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference %store: %w", ref.ReferenceOrDefault(), err)
	}

	manifestJSON, err := content.FetchAll(ctx, layoutStore, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content for %s: %w", ref.ReferenceOrDefault(), err)
	}

	var m v1.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	if len(m.Layers) != 1 {
		return nil, fmt.Errorf("expected a single layer in the manifest, got %d", len(m.Layers))
	}

	return content.FetchAll(ctx, layoutStore, m.Layers[0])
}

func (r *Repository) Path(ctx context.Context) (string, error) {
	ref, err := parseReference(r.tag)
	if err != nil {
		return "", err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return "", err
	}

	desc, err := layoutStore.Resolve(ctx, ref.ReferenceOrDefault())
	if err != nil {
		return "", fmt.Errorf("failed to resolve reference %s: %w", ref.ReferenceOrDefault(), err)
	}

	manifestJSON, err := content.FetchAll(ctx, layoutStore, desc)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content for %s: %w", ref.ReferenceOrDefault(), err)
	}

	var m v1.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return "", fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	if len(m.Layers) != 1 {
		return "", fmt.Errorf("expected a single layer in the manifest, got %d", len(m.Layers))
	}

	layerDigest := m.Layers[0].Digest
	blobPath := filepath.Join(baseDir, repoName(ref), "blobs", layerDigest.Algorithm().String(), layerDigest.Encoded())

	return blobPath, nil
}

func (r *Repository) Save(ctx context.Context, manifestPath string) error {
	ref, err := parseReference(r.tag)
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

func (r *Repository) Push(ctx context.Context) error {
	ref, err := parseReference(r.tag)
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

func (r *Repository) Pull(ctx context.Context) error {
	ref, err := parseReference(r.tag)
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
