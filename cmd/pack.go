// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/repository"
)

const (
	packFileFlag      = "file"
	packFileShortFlag = "f"
	packTagFlag       = "tag"
	packTagShortFlag  = "t"
)

type PackOpts struct {
	filePath string
	tag      string
}

var packOpts PackOpts

func init() {
	rootCmd.AddCommand(packCmd)

	flag := packCmd.Flags()
	flag.StringVarP(&packOpts.filePath, packFileFlag, packFileShortFlag, "", "Path to the manifest file to pack")
	flag.StringVarP(&packOpts.tag, packTagFlag, packTagShortFlag, "", "OCI reference for the packed manifest (e.g., registry.example.com/repo:tag)")

	_ = packCmd.MarkFlagRequired(packFileFlag)
	_ = packCmd.MarkFlagRequired(packTagFlag)
}

// packCmd represents the pack command
var packCmd = &cobra.Command{
	Use:   "pack",
	Short: "Pack a Kubernetes manifest into an OCI image layout",
	Long: `Pack packages a single Kubernetes manifest file into an OCI (Open Container
Initiative) image layout format for distribution and versioning.

This command creates an OCI-compliant artifact that can be pushed to OCI-compatible
registries, enabling manifest versioning, distribution, and deployment using
standard container tooling.

The packed manifest is stored in OCI image layout format, allowing it to be:
- Pushed to any OCI-compatible registry (Docker Hub, GitHub Container Repository, etc.)
- Tagged and versioned like container images
- Pulled and deployed using standard OCI tools

Examples:
  # Pack a manifest file with a full OCI reference
  kubectl mft pack -f deployment.yaml -t registry.example.com/manifests/app:v1.0.0

  # Pack a manifest for local OCI layout (using localhost)
  kubectl mft pack -f app.yaml -t localhost/myapp:production-v2.1.0

  # Pack a manifest with Docker Hub reference
  kubectl mft pack -f service.yaml -t docker.io/myorg/manifests:latest`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPack(cmd.Context())
	},
}

func runPack(ctx context.Context) error {
	r := repository.NewRepository(packOpts.tag)
	return mft.Pack(ctx, r, packOpts.filePath)
}
