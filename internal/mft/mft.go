package mft

import (
	"fmt"
	"os"
	"path/filepath"
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
