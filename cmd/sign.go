// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/oci"
	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

type SignOpts struct {
	tag string
	key string
}

var signOpts SignOpts

func init() {
	rootCmd.AddCommand(signCmd)

	flag := signCmd.Flags()
	flag.StringVar(&signOpts.key, "key", "default", "Name of the private key to use for signing")
}

// signCmd represents the sign command
var signCmd = &cobra.Command{
	Use:   "sign <tag>",
	Short: "Sign a packed manifest",
	Long: `Sign a previously packed manifest in local OCI layout storage.

The signing key must be generated first using 'kubectl mft key generate'.

Examples:
  # Sign a local manifest
  kubectl mft sign myapp:v1.0.0

  # Sign a manifest with registry reference
  kubectl mft sign registry.example.com/manifests/app:v1.0.0`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		signOpts.tag = args[0]
		return runSign(cmd.Context())
	},
}

func runSign(ctx context.Context) error {
	if !signature.PrivateKeyExists(signOpts.key) {
		return fmt.Errorf("signing key %q not found, run 'kubectl mft key generate' to create a key pair", signOpts.key)
	}

	r, err := oci.NewRepository(signOpts.tag)
	if err != nil {
		return err
	}

	signer, err := signature.NewSignerFromKeyDir(signOpts.key)
	if err != nil {
		return err
	}

	result, err := signer.Sign(ctx, r.LayoutPath(), r.Tag())
	if err != nil {
		return fmt.Errorf("failed to sign manifest: %w", err)
	}

	fmt.Printf("Signed %s (signature digest: %s)\n", r.Tag(), result.Digest)
	return nil
}
