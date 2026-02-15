// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
	"github.com/chez-shanpu/kubectl-mft/internal/validate"
)

type PackOpts struct {
	filePath       string
	tag            string
	skipValidation bool
}

var packOpts PackOpts

func init() {
	rootCmd.AddCommand(packCmd)

	flag := packCmd.Flags()
	flag.StringVarP(&packOpts.filePath, FileFlag, FileShortFlag, "", "Path to the manifest file to pack")
	flag.BoolVar(&packOpts.skipValidation, "skip-validation", false, "Skip manifest validation before packing")

	_ = packCmd.MarkFlagRequired(FileFlag)
}

// packCmd represents the pack command
var packCmd = &cobra.Command{
	Use:   "pack <tag>",
	Short: "Save a Kubernetes manifest into an OCI image layout",
	Long: `Save packages a single Kubernetes manifest file into an OCI (Open Container
Initiative) image layout format for distribution and versioning.

This command creates an OCI-compliant artifact that can be pushed to OCI-compatible
registries, enabling manifest versioning, distribution, and deployment using
standard container tooling.

The packed manifest is stored in OCI image layout format, allowing it to be:
- Pushed to any OCI-compatible registry (Docker Hub, GitHub Container Repository, etc.)
- Tagged and versioned like container images
- Pulled and deployed using standard OCI tools

Examples:
  # Save a manifest file with a full OCI reference
  kubectl mft pack -f deployment.yaml registry.example.com/manifests/app:v1.0.0

  # Save a manifest for local OCI layout (using localhost)
  kubectl mft pack -f app.yaml localhost/myapp:production-v2.1.0

  # Save a manifest with Docker Hub reference
  kubectl mft pack -f service.yaml docker.io/myorg/manifests:latest`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packOpts.tag = args[0]
		return runPack(cmd.Context())
	},
}

func runPack(ctx context.Context) error {
	if !packOpts.skipValidation {
		tmpl, err := validate.SchemaLocationTemplate()
		if err != nil {
			return fmt.Errorf("failed to resolve schema directory: %w", err)
		}
		if err := validate.ValidateManifest(packOpts.filePath,
			validate.WithSchemaLocations(tmpl),
		); err != nil {
			return fmt.Errorf("manifest validation failed: %w", err)
		}
	}

	r, err := oci.NewRepository(packOpts.tag)
	if err != nil {
		return err
	}
	return mft.Save(ctx, r, packOpts.filePath)
}
