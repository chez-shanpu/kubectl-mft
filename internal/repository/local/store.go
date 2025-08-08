// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package local

import (
	"fmt"
	"os"
	"path/filepath"

	"oras.land/oras-go/v2/content/oci"
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

// CreateOCILayoutStore creates a new OCI layout store at the specified path for the given reference
func CreateOCILayoutStore(dst string) (*oci.Store, error) {
	layoutPath := filepath.Join(baseDir, dst)
	layoutStore, err := oci.New(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create oci-layout store: %w", err)
	}
	return layoutStore, nil
}
