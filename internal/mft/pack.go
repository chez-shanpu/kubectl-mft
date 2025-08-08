// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"context"
	"fmt"
	"os"

	"github.com/chez-shanpu/kubectl-mft/internal/repository"
	"github.com/chez-shanpu/kubectl-mft/internal/repository/local"
)


// Pack packages a Kubernetes manifest into OCI layout format
func Pack(ctx context.Context, manifest string, tag string) error {
	ref, err := repository.ParseReference(tag)
	if err != nil {
		return err
	}

	manifestContent, err := prepareManifestContent(ctx, manifest, ref)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := manifestContent.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close manifest content: %v\n", closeErr)
		}
	}()

	layoutStore, err := local.CreateOCILayoutStore(manifestName(ref))
	if err != nil {
		return err
	}

	return repository.Copy(ctx, manifestContent.FileStore, layoutStore, ref)
}


