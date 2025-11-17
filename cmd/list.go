// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
)

type ListOpts struct {
	output string
}

var listOpts ListOpts

func init() {
	rootCmd.AddCommand(listCmd)

	flag := listCmd.Flags()
	flag.StringVarP(&listOpts.output, OutputFlag, OutputShortFlag, "table", "Output format (table, json, yaml)")
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all manifests in local OCI layout storage",
	Long: `List retrieves and displays all Kubernetes manifests stored in local OCI layout.

This command scans the local OCI layout directory and shows information about all
stored manifests including their repository names, tags, sizes, and creation times.

Output formats:
  - table: Human-readable table format (default)
  - json:  JSON format
  - yaml:  YAML format

Examples:
  # List all manifests in table format
  kubectl mft list

  # List in JSON format
  kubectl mft list -o json

  # List in YAML format
  kubectl mft list --output yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd.Context())
	},
}

func runList(ctx context.Context) error {
	r := oci.NewRegistry()
	res, err := mft.List(ctx, r)
	if err != nil {
		return err
	}

	res.Sort()
	return res.Print(mft.ListOutput(listOpts.output))
}
