// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/validate"
)

func init() {
	schemaCmd.AddCommand(schemaDeleteCmd)
}

// schemaDeleteCmd represents the schema delete command
var schemaDeleteCmd = &cobra.Command{
	Use:   "delete <group/kind>",
	Short: "Delete a registered CRD schema",
	Long: `Delete a registered CRD schema from local storage.

All versions of the specified resource schema will be removed.

Examples:
  # Delete a CRD schema
  kubectl mft schema delete cilium.io/CiliumNetworkPolicy

  # Delete another CRD schema
  kubectl mft schema delete cert-manager.io/Certificate`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSchemaDelete(args[0])
	},
}

func runSchemaDelete(groupKind string) error {
	group, kind, err := validate.ParseGroupKind(groupKind)
	if err != nil {
		return err
	}

	if err := validate.DeleteSchema(group, kind); err != nil {
		return err
	}
	fmt.Printf("CRD schema %s/%s deleted successfully\n", group, kind)
	return nil
}
