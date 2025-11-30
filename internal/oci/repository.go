// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
)

const (
	artifactType     = "application/vnd.kubectl-mft.v1"
	contentMediaType = "application/vnd.kubectl-mft.content.v1+yaml"

	// DefaultRegistry is the default registry name used for simple tag names without a slash
	DefaultRegistry = "local"
)

const (
	workingDIR = "/tmp/kubectl-mft"
)

var baseDir string

func init() {
	// Check for environment variable first (useful for testing)
	if dir := os.Getenv("KUBECTL_MFT_STORAGE_DIR"); dir != "" {
		baseDir = dir
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get user home directory: %v\n", err)
		os.Exit(1)
	}
	baseDir = filepath.Join(home, ".local", "share", "kubectl-mft", "manifests")
}

type Repository struct {
	ref *registry.Reference
}

func NewRepository(tag string) (*Repository, error) {
	ref, err := parseReference(tag)
	if err != nil {
		return nil, err
	}

	return &Repository{ref: ref}, nil
}

func (r *Repository) Copy(ctx context.Context, dest string) error {
	drepo, err := NewRepository(dest)
	if err != nil {
		return fmt.Errorf("creating repository: %w", err)
	}

	sstore, err := r.newOCILayoutStore()
	if err != nil {
		return err
	}

	// Check source exists
	_, err = sstore.Resolve(ctx, r.ref.ReferenceOrDefault())
	if err != nil {
		if errors.Is(err, errdef.ErrNotFound) {
			return fmt.Errorf("source tag %q not found in local storage", r.ref.ReferenceOrDefault())
		} else {
			return fmt.Errorf("failed to resolve source tag: %w", err)
		}
	}

	destStore, err := drepo.newOCILayoutStore()
	if err != nil {
		return err
	}

	_, err = destStore.Resolve(ctx, drepo.ref.ReferenceOrDefault())
	if err == nil {
		return fmt.Errorf("destination tag %q already exists", drepo.ref.ReferenceOrDefault())
	}
	if !errors.Is(err, errdef.ErrNotFound) {
		return fmt.Errorf("failed to check destination tag: %w", err)
	}

	_, err = oras.Copy(ctx, sstore, r.ref.ReferenceOrDefault(), destStore, drepo.ref.ReferenceOrDefault(), oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to copy manifest: %w", err)
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context) (*mft.DeleteResult, error) {
	layoutStore, err := r.newOCILayoutStore()
	if err != nil {
		return nil, err
	}

	desc, err := layoutStore.Resolve(ctx, r.ref.ReferenceOrDefault())
	if err != nil {
		// If not found, return nil (idempotent behavior)
		if errors.Is(err, errdef.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve reference %s: %w", r.ref.ReferenceOrDefault(), err)
	}

	if err := layoutStore.Delete(ctx, desc); err != nil {
		return nil, fmt.Errorf("failed to delete manifest: %w", err)
	}

	indexDir := filepath.Join(baseDir, r.Name())
	if err := deleteRepositoryIfEmpty(indexDir); err != nil {
		return nil, fmt.Errorf("failed to delete repository: %w", err)
	}

	return mft.NewDeleteResult(
		r.Name(),
		r.ref.ReferenceOrDefault(),
	), nil
}

func (r *Repository) Dump(ctx context.Context) (*mft.DumpResult, error) {
	layoutStore, err := r.newOCILayoutStore()
	if err != nil {
		return nil, err
	}

	desc, err := layoutStore.Resolve(ctx, r.ref.ReferenceOrDefault())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference %s: %w", r.ref.ReferenceOrDefault(), err)
	}

	manifestJSON, err := content.FetchAll(ctx, layoutStore, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content for %s: %w", r.ref.ReferenceOrDefault(), err)
	}

	var m v1.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	if len(m.Layers) != 1 {
		return nil, fmt.Errorf("expected a single layer in the manifest, got %d", len(m.Layers))
	}

	b, err := content.FetchAll(ctx, layoutStore, m.Layers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content for %s: %w", r.ref.ReferenceOrDefault(), err)
	}
	return mft.NewDumpResult(b), nil
}

func (r *Repository) Path(ctx context.Context) (*mft.PathResult, error) {
	layoutStore, err := r.newOCILayoutStore()
	if err != nil {
		return nil, err
	}

	desc, err := layoutStore.Resolve(ctx, r.ref.ReferenceOrDefault())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference %s: %w", r.ref.ReferenceOrDefault(), err)
	}

	manifestJSON, err := content.FetchAll(ctx, layoutStore, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content for %s: %w", r.ref.ReferenceOrDefault(), err)
	}

	var m v1.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	if len(m.Layers) != 1 {
		return nil, fmt.Errorf("expected a single layer in the manifest, got %d", len(m.Layers))
	}

	layerDigest := m.Layers[0].Digest
	blobPath := filepath.Join(baseDir, r.Name(), "blobs", layerDigest.Algorithm().String(), layerDigest.Encoded())

	return mft.NewPathResult(blobPath), nil
}

func (r *Repository) Pull(ctx context.Context) error {
	layoutStore, err := r.newOCILayoutStore()
	if err != nil {
		return err
	}

	repo, err := r.newAuthenticatedRepository()
	if err != nil {
		return err
	}

	return r.copy(ctx, repo, layoutStore)
}

func (r *Repository) Push(ctx context.Context) error {
	layoutStore, err := r.newOCILayoutStore()
	if err != nil {
		return err
	}

	repo, err := r.newAuthenticatedRepository()
	if err != nil {
		return err
	}

	return r.copy(ctx, layoutStore, repo)
}

func (r *Repository) Save(ctx context.Context, manifestPath string) (err error) {
	fs, err := r.newFileStore(ctx, manifestPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fs.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("warning: failed to close manifestPath content: %w", err)
		}
	}()

	layoutStore, err := r.newOCILayoutStore()
	if err != nil {
		return err
	}

	return r.copy(ctx, fs, layoutStore)
}

func (r *Repository) Name() string {
	if r.ref.Registry != "" && r.ref.Repository != "" {
		return r.ref.Registry + "/" + r.ref.Repository
	} else if r.ref.Registry != "" {
		return r.ref.Registry
	} else if r.ref.Repository != "" {
		return r.ref.Repository
	}
	return ""
}

// copyRepo handles copying manifests between OCI targets
func (r *Repository) copy(ctx context.Context, source oras.Target, dest oras.Target) error {
	_, err := oras.Copy(ctx, source, r.ref.ReferenceOrDefault(), dest, r.ref.ReferenceOrDefault(), oras.DefaultCopyOptions)
	if err != nil {
		return r.formatCopyError(err)
	}
	return nil
}

// formatCopyError provides better error messages based on common registry operation failures
func (r *Repository) formatCopyError(err error) error {
	if err == nil {
		return fmt.Errorf("unknown error occurred with %s/%s:%s",
			r.ref.Registry, r.ref.Repository, r.ref.ReferenceOrDefault())
	}

	errorMsg := err.Error()

	// Check for common error patterns and provide helpful messages
	if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") {
		return fmt.Errorf("authentication failed for registry %s: %w\n"+
			"Please ensure you are logged in using 'docker login %s'", r.ref.Registry, err, r.ref.Registry)
	}

	if strings.Contains(errorMsg, "403") || strings.Contains(errorMsg, "forbidden") {
		return fmt.Errorf("access denied to repository %s/%s: %w\n"+
			"Check if you have the required permissions to this repository", r.ref.Registry, r.ref.Repository, err)
	}

	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "network") {
		return fmt.Errorf("network error with %s: %w\n"+
			"Check your network connection and registry availability", r.ref.Registry, err)
	}

	return fmt.Errorf("failed to copy manifest %s/%s:%s: %w",
		r.ref.Registry, r.ref.Repository, r.ref.ReferenceOrDefault(), err)
}

// newAuthenticatedRepository creates and configures a repository with authentication
func (r *Repository) newAuthenticatedRepository() (*remote.Repository, error) {
	c, err := newCredentialFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential for registry %s: %w", r.ref.Registry, err)
	}

	repo, err := remote.NewRepository(filepath.Join(r.ref.Registry, r.ref.Repository))
	if err != nil {
		return nil, fmt.Errorf("failed to create repository %s/%s: %w", r.ref.Registry, r.ref.Repository, err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: c,
	}

	// Enable PlainHTTP for localhost registries (for testing)
	if isLocalRegistry(r.ref.Registry) {
		repo.PlainHTTP = true
	}

	return repo, nil
}

func (r *Repository) newFileStore(ctx context.Context, manifestPath string) (*file.Store, error) {
	// Clean up working directory to ensure a fresh start for each operation
	if err := os.RemoveAll(workingDIR); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to clean working directory: %w", err)
	}

	fs, err := file.New(workingDIR)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %w", err)
	}

	path, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of %q: %w", manifestPath, err)
	}

	// Use tag-specific Name to avoid duplicates within the same file store
	contentName := fmt.Sprintf("%s:%s", r.Name(), r.ref.ReferenceOrDefault())
	contentDesc, err := fs.Add(ctx, contentName, contentMediaType, path)
	if err != nil {
		return nil, fmt.Errorf("failed to add content: %w", err)
	}

	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers: []v1.Descriptor{contentDesc},
		ManifestAnnotations: map[string]string{
			"org.opencontainers.image.title": r.Name(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pack manifestPath: %w", err)
	}

	// Tag the manifestPath
	tagRef := r.ref.ReferenceOrDefault()
	if err = fs.Tag(ctx, manifestDesc, tagRef); err != nil {
		return nil, fmt.Errorf("failed to tag manifestPath: %w", err)
	}

	return fs, nil
}

func (r *Repository) newOCILayoutStore() (*oci.Store, error) {
	layoutPath := filepath.Join(baseDir, r.Name())
	layoutStore, err := oci.New(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create oci-layout store: %w", err)
	}
	return layoutStore, nil
}

func deleteRepositoryIfEmpty(indexDir string) error {
	indexData, err := os.ReadFile(filepath.Join(indexDir, "index.json"))
	if err != nil {
		return fmt.Errorf("failed to read index.json: %w", err)
	}

	var index *v1.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		return fmt.Errorf("failed to unmarshal index.json: %w", err)
	}

	if len(index.Manifests) != 0 {
		return nil // Repository is not empty
	}

	if err := os.RemoveAll(indexDir); err != nil {
		return fmt.Errorf("warning: failed to remove repository directory: %w", err)
	}

	return nil
}

// parseReference parses and validates the OCI reference.
// If the tag doesn't contain a slash, it prepends the default registry name.
func parseReference(tag string) (*registry.Reference, error) {
	normalizedTag := normalizeTag(tag)
	ref, err := registry.ParseReference(normalizedTag)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference %q: %w", tag, err)
	}
	return &ref, nil
}

// normalizeTag adds the default registry prefix if the tag doesn't contain a slash.
// For example: "myapp:v1" becomes "local/myapp:v1"
func normalizeTag(tag string) string {
	if !strings.Contains(tag, "/") {
		return DefaultRegistry + "/" + tag
	}
	return tag
}

// isLocalRegistry checks if the registry is a local/test registry that should use PlainHTTP
func isLocalRegistry(registry string) bool {
	return strings.HasPrefix(registry, "localhost") ||
		strings.HasPrefix(registry, "127.0.0.1")
}
