// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
)

const (
	artifactType     = "application/vnd.kubectl-mft.v1"
	contentMediaType = "application/vnd.kubectl-mft.content.v1+yaml"
)

const (
	workingDIR = "/tmp/kubectl-mft"
)

var ociDIR string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get user home directory: %v\n", err)
		os.Exit(1)
	}
	ociDIR = filepath.Join(home, ".local", "share", "kubectl-mft", "manifests")
}

// parseReference parses and validates the OCI reference
func parseReference(tag string) (registry.Reference, error) {
	ref, err := registry.ParseReference(tag)
	if err != nil {
		return registry.Reference{}, fmt.Errorf("failed to parse reference %q: %w", tag, err)
	}
	return ref, nil
}

// createOCILayoutStore creates a new OCI layout store at the specified path for the given reference
func createOCILayoutStore(ref *registry.Reference) (*oci.Store, error) {
	layoutPath := filepath.Join(ociDIR, manifestDIRName(ref))
	layoutStore, err := oci.New(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create oci-layout store: %w", err)
	}
	return layoutStore, nil
}

// manifestDIRName generates a directory name for the manifest based on OCI reference
// Format: <registry>-<repository>-<tag>, where "/" in the repository is replaced with "-"
// Example: "docker.io-user/app-v1.0.0" becomes "docker.io-user-app-v1.0.0"
func manifestDIRName(r *registry.Reference) string {
	s := []string{r.Registry, strings.ReplaceAll(r.Repository, "/", "-"), r.ReferenceOrDefault()}
	return strings.Join(s, "-")
}
