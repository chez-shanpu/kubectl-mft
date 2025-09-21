// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	artifactType     = "application/vnd.kubectl-mft.v1"
	contentMediaType = "application/vnd.kubectl-mft.content.v1+yaml"
)

const (
	workingDIR = "/tmp/kubectl-mft"
)

var baseDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get user home directory: %v\n", err)
		os.Exit(1)
	}
	baseDir = filepath.Join(home, ".local", "share", "kubectl-mft", "manifests")
}

// copyRepo handles copying manifests between OCI targets
func copyRepo(ctx context.Context, source oras.Target, dest oras.Target, ref *registry.Reference) error {
	_, err := oras.Copy(ctx, source, ref.ReferenceOrDefault(), dest, ref.ReferenceOrDefault(), oras.DefaultCopyOptions)
	if err != nil {
		return formatRegistryError(err, ref)
	}
	return nil
}

// formatRegistryError provides better error messages based on common registry operation failures
func formatRegistryError(err error, ref *registry.Reference) error {
	if err == nil {
		return fmt.Errorf("unknown error occurred with %s/%s:%s",
			ref.Registry, ref.Repository, ref.ReferenceOrDefault())
	}

	errorMsg := err.Error()

	// Check for common error patterns and provide helpful messages
	if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") {
		return fmt.Errorf("authentication failed for registry %s: %w\n"+
			"Please ensure you are logged in using 'docker login %s'", ref.Registry, err, ref.Registry)
	}

	if strings.Contains(errorMsg, "403") || strings.Contains(errorMsg, "forbidden") {
		return fmt.Errorf("access denied to repository %s/%s: %w\n"+
			"Check if you have the required permissions to this repository", ref.Registry, ref.Repository, err)
	}

	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "network") {
		return fmt.Errorf("network error with %s: %w\n"+
			"Check your network connection and registry availability", ref.Registry, err)
	}

	return fmt.Errorf("failed to access manifest at %s/%s:%s: %w",
		ref.Registry, ref.Repository, ref.ReferenceOrDefault(), err)
}

// newAuthenticatedRepository creates and configures a repository with authentication
func newAuthenticatedRepository(ref *registry.Reference) (*remote.Repository, error) {
	c, err := newCredentialFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential for registry %s: %w", ref.Registry, err)
	}

	repo, err := remote.NewRepository(filepath.Join(ref.Registry, ref.Repository))
	if err != nil {
		return nil, fmt.Errorf("failed to create repository %s/%s: %w", ref.Registry, ref.Repository, err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: c,
	}

	return repo, nil
}

func newFileStore(ctx context.Context, ref *registry.Reference, manifestPath string) (*file.Store, error) {
	fs, err := file.New(workingDIR)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %w", err)
	}

	path, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of %q: %w", manifestPath, err)
	}

	name := strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))
	contentDesc, err := fs.Add(ctx, name, contentMediaType, path)
	if err != nil {
		return nil, fmt.Errorf("failed to add content: %w", err)
	}

	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers: []v1.Descriptor{contentDesc},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pack manifestPath: %w", err)
	}

	// Tag the manifestPath
	tagRef := ref.ReferenceOrDefault()
	if err = fs.Tag(ctx, manifestDesc, tagRef); err != nil {
		return nil, fmt.Errorf("failed to tag manifestPath: %w", err)
	}

	return fs, nil
}

func newOCILayoutStore(ref *registry.Reference) (*oci.Store, error) {
	layoutPath := filepath.Join(baseDir, repoDIR(ref))
	layoutStore, err := oci.New(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create oci-layout store: %w", err)
	}
	return layoutStore, nil
}

// parseReference parses and validates the OCI reference
func parseReference(tag string) (*registry.Reference, error) {
	ref, err := registry.ParseReference(tag)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference %q: %w", tag, err)
	}
	return &ref, nil
}

func repoDIR(r *registry.Reference) string {
	s := []string{r.Registry, strings.ReplaceAll(r.Repository, "/", "-"), r.ReferenceOrDefault()}
	return strings.Join(s, "-")
}
