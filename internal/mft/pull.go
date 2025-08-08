// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"context"

	"github.com/chez-shanpu/kubectl-mft/internal/repository"
	"github.com/chez-shanpu/kubectl-mft/internal/repository/local"
	"github.com/chez-shanpu/kubectl-mft/internal/repository/remote"
)

// Pull pulls a Kubernetes manifest from an OCI registry
func Pull(ctx context.Context, tag string) error {
	ref, err := repository.ParseReference(tag)
	if err != nil {
		return err
	}

	layoutStore, err := local.CreateOCILayoutStore(manifestName(ref))
	if err != nil {
		return err
	}

	repo, err := remote.CreateAuthenticatedRepository(ref)
	if err != nil {
		return err
	}

	return repository.Copy(ctx, repo, layoutStore, ref)
}
