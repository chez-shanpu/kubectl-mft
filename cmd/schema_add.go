// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/validate"
)

type SchemaAddOpts struct {
	filePath string
}

var schemaAddOpts SchemaAddOpts

func init() {
	schemaCmd.AddCommand(schemaAddCmd)

	flag := schemaAddCmd.Flags()
	flag.StringVarP(&schemaAddOpts.filePath, FileFlag, FileShortFlag, "", "Path to the CRD YAML file")

	_ = schemaAddCmd.MarkFlagRequired(FileFlag)
}

// schemaAddCmd represents the schema add command
var schemaAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Register a CRD schema for custom resource validation",
	Long: `Register a CustomResourceDefinition (CRD) schema for use during manifest validation.

The command reads the CRD YAML file, extracts the OpenAPI v3 schema from each version,
and stores it locally for use during pack validation.

Examples:
  # Register a CRD schema from a file
  kubectl mft schema add -f ciliumnetworkpolicy-crd.yaml

  # Register a CRD schema from a downloaded file
  kubectl mft schema add -f cert-manager-certificate-crd.yaml`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSchemaAdd()
	},
}

func runSchemaAdd() error {
	if err := validate.RegisterCRDSchema(schemaAddOpts.filePath); err != nil {
		return err
	}
	fmt.Println("CRD schema registered successfully")
	return nil
}
