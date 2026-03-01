// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

func init() {
	keyCmd.AddCommand(keyListCmd)
}

// keyListCmd represents the key list command
var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all signing keys",
	Long: `List all keys stored in the key directory.

Examples:
  kubectl mft key list`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyList()
	},
}

func runKeyList() error {
	keys, err := signature.ListKeys()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println("No keys found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tPATH")
	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s\t%s\n", k.Name, k.Type, k.Path)
	}
	return w.Flush()
}
