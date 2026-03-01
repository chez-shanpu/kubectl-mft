// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
	"github.com/chez-shanpu/kubectl-mft/internal/signature"
	"github.com/chez-shanpu/kubectl-mft/internal/validate"
)

type PackOpts struct {
	filePath       string
	tag            string
	skipValidation bool
	skipSign       bool
	key            string
}

var packOpts PackOpts

func init() {
	rootCmd.AddCommand(packCmd)

	flag := packCmd.Flags()
	flag.StringVarP(&packOpts.filePath, FileFlag, FileShortFlag, "", "Path to the manifest file to pack")
	flag.BoolVar(&packOpts.skipValidation, "skip-validation", false, "Skip manifest validation before packing")
	flag.BoolVar(&packOpts.skipSign, "skip-sign", false, "Skip signing the packed manifest")
	flag.StringVar(&packOpts.key, "key", "default", "Name of the private key to use for signing")

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

	// Check signing key before saving to avoid partial state
	if !packOpts.skipSign {
		if !signature.PrivateKeyExists(packOpts.key) {
			return fmt.Errorf("signing key %q not found, run 'kubectl mft key generate' to create a key pair, or use '--skip-sign' to skip signing", packOpts.key)
		}
	}

	r, err := oci.NewRepository(packOpts.tag)
	if err != nil {
		return err
	}
	if err := mft.Save(ctx, r, packOpts.filePath); err != nil {
		return err
	}

	if !packOpts.skipSign {
		signer, err := signature.NewSignerFromKeyDir(packOpts.key)
		if err != nil {
			return deletePackedData(ctx, r, err)
		}
		if _, err := signer.Sign(ctx, r.LayoutPath(), r.Tag()); err != nil {
			return deletePackedData(ctx, r, fmt.Errorf("failed to sign manifest: %w", err))
		}
	}

	return nil
}

func deletePackedData(ctx context.Context, r *oci.Repository, originalErr error) error {
	if _, deleteErr := mft.Delete(ctx, r); deleteErr != nil {
		return errors.Join(originalErr, fmt.Errorf("failed to clean up packed data: %w", deleteErr))
	}
	return originalErr
}
