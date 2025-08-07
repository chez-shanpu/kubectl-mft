package mft

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// Push pushes a Kubernetes manifest to an OCI registry
func Push(ctx context.Context, tag string) error {
	ref, err := parseReference(tag)
	if err != nil {
		return err
	}

	layoutStore, err := createOCILayoutStore(&ref)
	if err != nil {
		return err
	}

	repo, err := createAuthenticatedRepository(&ref)
	if err != nil {
		return err
	}

	return pushToRepository(ctx, layoutStore, repo, &ref)
}

// createAuthenticatedRepository creates and configures a repository with authentication
func createAuthenticatedRepository(ref *registry.Reference) (*remote.Repository, error) {
	repo, err := remote.NewRepository(filepath.Join(ref.Registry, ref.Repository))
	if err != nil {
		return nil, fmt.Errorf("failed to create repository %s/%s: %w", ref.Registry, ref.Repository, err)
	}

	credStore, err := createCredentialStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential store for registry %s: %w", ref.Registry, err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
	}

	return repo, nil
}

// createCredentialStore creates a credential store with secure defaults
func createCredentialStore() (credentials.Store, error) {
	opt := credentials.StoreOptions{
		AllowPlaintextPut: false, // Secure default
	}
	s, err := credentials.NewStoreFromDocker(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential store: %w", err)
	}
	return s, nil
}

// pushToRepository handles the actual push operation to the repository
func pushToRepository(ctx context.Context, source oras.Target, dest oras.Target, ref *registry.Reference) error {
	_, err := oras.Copy(ctx, source, ref.ReferenceOrDefault(), dest, ref.ReferenceOrDefault(), oras.DefaultCopyOptions)
	if err != nil {
		return formatPushError(err, ref)
	}
	return nil
}

// formatPushError provides better error messages based on common push failures
func formatPushError(err error, ref *registry.Reference) error {
	errorMsg := err.Error()

	// Check for common error patterns and provide helpful messages
	if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") {
		return fmt.Errorf("authentication failed for registry %s: %w\n"+
			"Please ensure you are logged in using 'docker login %s'", ref.Registry, err, ref.Registry)
	}

	if strings.Contains(errorMsg, "403") || strings.Contains(errorMsg, "forbidden") {
		return fmt.Errorf("access denied to repository %s/%s: %w\n"+
			"Check if you have push permissions to this repository", ref.Registry, ref.Repository, err)
	}

	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "network") {
		return fmt.Errorf("network error while pushing to %s: %w\n"+
			"Check your network connection and registry availability", ref.Registry, err)
	}

	return fmt.Errorf("failed to push manifest to %s/%s:%s: %w",
		ref.Registry, ref.Repository, ref.ReferenceOrDefault(), err)
}
