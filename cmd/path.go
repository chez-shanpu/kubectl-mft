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

	flag := pathCmd.Flags()
	flag.StringVarP(&pathOpts.tag, TagFlag, TagShortFlag, "", "OCI reference for the manifest (e.g., registry.example.com/repo:tag)")

	_ = pathCmd.MarkFlagRequired(TagFlag)
}

// pathCmd represents the path command
var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Get the file system path to a manifest in local OCI layout storage",
	Long: `Path retrieves the file system path to a Kubernetes manifest stored in local OCI layout.

This command returns the absolute file path to the manifest blob in the OCI layout directory.
The manifest must have been previously packed using the 'pack' command or pulled using the 'pull' command.

Examples:
  # Get the path to a manifest
  kubectl mft path -t registry.example.com/manifests/app:v1.0.0

  # Use with kubectl debug --custom option
  kubectl debug my-pod --custom $(kubectl mft path -t localhost/debug-container:latest)`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
