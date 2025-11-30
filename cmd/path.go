// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
)

type PathOpts struct {
	tag string
}

var pathOpts PathOpts

func init() {
	rootCmd.AddCommand(pathCmd)
}

// pathCmd represents the path command
var pathCmd = &cobra.Command{
	Use:   "path <tag>",
	Short: "Get the file system path to a manifest in local OCI layout storage",
	Long: `Path retrieves the file system path to a Kubernetes manifest stored in local OCI layout.

This command returns the absolute file path to the manifest blob in the OCI layout directory.
The manifest must have been previously packed using the 'pack' command or pulled using the 'pull' command.

Examples:
  # Get the path to a manifest
  kubectl mft path registry.example.com/manifests/app:v1.0.0

  # Use with kubectl debug --custom option
  kubectl debug my-pod --custom $(kubectl mft path localhost/debug-container:latest)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pathOpts.tag = args[0]
		return runPath(cmd.Context())
	},
}

func runPath(ctx context.Context) error {
	r, err := oci.NewRepository(pathOpts.tag)
	if err != nil {
		return err
	}

	res, err := mft.Path(ctx, r)
	if err != nil {
		return err
	}
	res.Print()
	return nil
}
