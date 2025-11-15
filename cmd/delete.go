// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
)

type DeleteOpts struct {
	tag   string
	force bool
}

var deleteOpts DeleteOpts

func init() {
	rootCmd.AddCommand(deleteCmd)

	flag := deleteCmd.Flags()
	flag.StringVarP(&deleteOpts.tag, TagFlag, TagShortFlag, "", "OCI reference for the manifest to delete (e.g., registry.example.com/repo:tag)")
	flag.BoolVarP(&deleteOpts.force, ForceFlag, ForceShortFlag, false, "Skip confirmation prompt")

	_ = deleteCmd.MarkFlagRequired(TagFlag)
}

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a manifest from local OCI layout storage",
	Long: `Delete removes a Kubernetes manifest from local OCI layout storage.

This command deletes a previously stored manifest from the local OCI layout.
Orphaned blobs (blobs only referenced by the deleted manifest) are automatically removed.
If the deleted manifest is the last one in the repository, the entire repository directory is removed.

By default, a confirmation prompt is shown before deletion. Use the --force flag to skip confirmation.

Examples:
  # Delete a manifest with confirmation
  kubectl mft delete -t registry.example.com/manifests/app:v1.0.0

  # Delete without confirmation
  kubectl mft delete -t localhost/myapp:latest --force

  # Delete with verbose output
  kubectl mft delete -t localhost/myapp:latest -v

  # Delete quietly (no output on success)
  kubectl mft delete -t localhost/myapp:latest -q`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDelete(cmd.Context())
	},
}

func runDelete(ctx context.Context) error {
	r, err := oci.NewRepository(deleteOpts.tag)
	if err != nil {
		return err
	}

	if !deleteOpts.force {
		if !confirmDeletion(deleteOpts.tag) {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	res, err := mft.Delete(ctx, r)
	if err != nil {
		return err
	}
	if res == nil {
		fmt.Printf("Warning: manifest %s not found locally\n", deleteOpts.tag)
		return nil
	}

	res.Print()
	return nil
}

// confirmDeletion shows a confirmation prompt and returns true if user confirms
func confirmDeletion(tag string) bool {
	fmt.Printf("Delete manifest %s? (y/N): ", tag)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
