// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/validate"
)

func init() {
	schemaCmd.AddCommand(schemaListCmd)
}

// schemaListCmd represents the schema list command
var schemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered CRD schemas",
	Long: `List all CRD schemas registered for manifest validation.

Examples:
  kubectl mft schema list`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSchemaList()
	},
}

func runSchemaList() error {
	schemas, err := validate.ListSchemas()
	if err != nil {
		return err
	}

	if len(schemas) == 0 {
		fmt.Println("No CRD schemas registered")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "GROUP\tKIND\tVERSION")
	for _, s := range schemas {
		fmt.Fprintf(w, "%s\t%s\t%s\n", s.Group, s.Kind, s.Version)
	}
	return w.Flush()
}
