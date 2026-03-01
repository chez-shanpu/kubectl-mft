// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

type KeyGenerateOpts struct {
	name  string
	force bool
}

var keyGenerateOpts KeyGenerateOpts

func init() {
	keyCmd.AddCommand(keyGenerateCmd)

	flag := keyGenerateCmd.Flags()
	flag.StringVar(&keyGenerateOpts.name, "name", "default", "Name for the key pair")
	flag.BoolVar(&keyGenerateOpts.force, ForceFlag, false, "Overwrite existing key pair")
}

// keyGenerateCmd represents the key generate command
var keyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate an ECDSA P-256 key pair for signing",
	Long: `Generate an ECDSA P-256 key pair and store it in the key directory.

The private key is saved as <name>.key and the public key as <name>.pub.
Share the public key with others for signature verification.

Examples:
  # Generate with default name
  kubectl mft key generate

  # Generate with custom name
  kubectl mft key generate --name mykey

  # Overwrite existing key pair
  kubectl mft key generate --force`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeyGenerate()
	},
}

func runKeyGenerate() error {
	if err := signature.GenerateKeyPair(keyGenerateOpts.name, keyGenerateOpts.force); err != nil {
		return err
	}

	privPath := signature.PrivateKeyPath(keyGenerateOpts.name)
	pubPath := signature.PublicKeyPath(keyGenerateOpts.name)
	fmt.Printf("Key pair generated successfully\nPrivate key: %s\nPublic key:  %s\nShare the public key with others for signature verification.\n",
		privPath, pubPath)
	return nil
}
