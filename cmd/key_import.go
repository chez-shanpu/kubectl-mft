// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

type KeyImportOpts struct {
	name string
}

var keyImportOpts KeyImportOpts

func init() {
	keyCmd.AddCommand(keyImportCmd)

	flag := keyImportCmd.Flags()
	flag.StringVar(&keyImportOpts.name, "name", "", "Name for the imported public key (default: filename without extension)")
}

// keyImportCmd represents the key import command
var keyImportCmd = &cobra.Command{
	Use:   "import <public-key-file>",
	Short: "Import a public key for signature verification",
	Long: `Import a PEM-encoded public key file into the key directory.

The imported key will be used during signature verification when pulling manifests.

Examples:
  # Import a public key with auto-detected name
  kubectl mft key import alice.pub

  # Import with a custom name
  kubectl mft key import /path/to/key.pub --name alice`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyImport(args[0])
	},
}

func runKeyImport(srcPath string) error {
	if err := signature.ImportPublicKey(srcPath, keyImportOpts.name); err != nil {
		return err
	}
	fmt.Printf("Public key imported successfully\n")
	return nil
}
