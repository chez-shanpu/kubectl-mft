// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

type KeyExportOpts struct {
	name string
}

var keyExportOpts KeyExportOpts

func init() {
	keyCmd.AddCommand(keyExportCmd)

	flag := keyExportCmd.Flags()
	flag.StringVar(&keyExportOpts.name, "name", "default", "Name of the public key to export")
}

// keyExportCmd represents the key export command
var keyExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a public key to stdout",
	Long: `Export a PEM-encoded public key to stdout for sharing with others.

Examples:
  # Export the default public key
  kubectl mft key export

  # Export a named public key
  kubectl mft key export --name alice

  # Save to a file
  kubectl mft key export > mykey.pub`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyExport()
	},
}

func runKeyExport() error {
	data, err := signature.ExportPublicKey(keyExportOpts.name)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
