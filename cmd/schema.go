// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(schemaCmd)
}

// schemaCmd represents the schema command group
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage CRD schemas for manifest validation",
	Long: `Manage CustomResourceDefinition (CRD) schemas used for manifest validation.

Registered CRD schemas allow kubectl-mft to validate custom resources during packing,
catching errors before they reach the cluster.

Examples:
  # Register a CRD schema
  kubectl mft schema add -f ciliumnetworkpolicy-crd.yaml

  # List registered schemas
  kubectl mft schema list

  # Delete a registered schema
  kubectl mft schema delete cilium.io/CiliumNetworkPolicy`,
}
