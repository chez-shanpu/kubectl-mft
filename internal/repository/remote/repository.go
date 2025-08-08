// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package remote

import (
	"fmt"
	"path/filepath"

	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/chez-shanpu/kubectl-mft/internal/credential"
)

// CreateAuthenticatedRepository creates and configures a repository with authentication
func CreateAuthenticatedRepository(ref *registry.Reference) (*remote.Repository, error) {
	repo, err := remote.NewRepository(filepath.Join(ref.Registry, ref.Repository))
	if err != nil {
		return nil, fmt.Errorf("failed to create repository %s/%s: %w", ref.Registry, ref.Repository, err)
	}

	c, err := credential.CreateFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to create credential for registry %s: %w", ref.Registry, err)
	}

	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.NewCache(),
		Credential: c,
	}

	return repo, nil
}
