// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

type KeyDeleteOpts struct {
	private bool
}

var keyDeleteOpts KeyDeleteOpts

func init() {
	keyDeleteCmd.Flags().BoolVar(&keyDeleteOpts.private, "private", false, "Delete the private key instead of the public key")
	keyCmd.AddCommand(keyDeleteCmd)
}

// keyDeleteCmd represents the key delete command
var keyDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a public or private key",
	Long: `Delete a named key from the key directory.

By default, this command deletes the public key. Use --private to delete
the private key instead.

Examples:
  # Delete a public key
  kubectl mft key delete alice

  # Delete a private key
  kubectl mft key delete --private alice`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyDelete(args[0], keyDeleteOpts)
	},
}

func runKeyDelete(name string, opts KeyDeleteOpts) error {
	if opts.private {
		if err := signature.DeletePrivateKey(name); err != nil {
			return err
		}
		fmt.Printf("Private key %q deleted successfully\n", name)
		return nil
	}

	if err := signature.DeletePublicKey(name); err != nil {
		return err
	}
	fmt.Printf("Public key %q deleted successfully\n", name)
	return nil
}
