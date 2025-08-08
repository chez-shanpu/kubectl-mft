// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"strings"

	"oras.land/oras-go/v2/registry"
)

const (
	artifactType     = "application/vnd.kubectl-mft.v1"
	contentMediaType = "application/vnd.kubectl-mft.content.v1+yaml"
)

const (
	workingDIR = "/tmp/kubectl-mft"
)

// manifestName generates a directory name for the manifest based on OCI reference
// Format: <registry>-<repository>-<tag>, where "/" in the repository is replaced with "-"
// Example: "docker.io-user/app-v1.0.0" becomes "docker.io-user-app-v1.0.0"
func manifestName(r *registry.Reference) string {
	s := []string{r.Registry, strings.ReplaceAll(r.Repository, "/", "-"), r.ReferenceOrDefault()}
	return strings.Join(s, "-")
}
