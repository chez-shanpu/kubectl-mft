// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
)

func init() {
	rootCmd.AddCommand(cpCmd)
}

// cpCmd represents the cp command
var cpCmd = &cobra.Command{
	Use:   "cp <source-tag> <destination-tag>",
	Short: "Copy a manifest to a new tag",
	Long: `Copy a manifest from one tag to another in local storage.

This command performs a deep copy, duplicating both the manifest and its blobs.
You can copy across different registries or repositories within local storage.`,
	Args: cobra.ExactArgs(2),
	RunE: runCopy,
}

func runCopy(cmd *cobra.Command, args []string) error {
	src := args[0]
	dest := args[1]

	sourceRepo, err := oci.NewRepository(src)
	if err != nil {
		return err
	}

	return mft.Copy(cmd.Context(), sourceRepo, dest)
}
