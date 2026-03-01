// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(keyCmd)
}

// keyCmd represents the key command group
var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage signing keys",
	Long: `Manage ECDSA P-256 signing keys for OCI artifact signing and verification.

Keys are stored in ~/.local/share/kubectl-mft/keys/ and used to sign
manifests during pack and verify signatures during pull.

Examples:
  # Generate a new key pair
  kubectl mft key generate

  # Import a public key for verification
  kubectl mft key import alice.pub --name alice

  # List all keys
  kubectl mft key list

  # Export a public key for sharing
  kubectl mft key export --name default

  # Delete a public key
  kubectl mft key delete alice`,
}
