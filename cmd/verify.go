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

type VerifyOpts struct {
	tag string
}

var verifyOpts VerifyOpts

func init() {
	rootCmd.AddCommand(verifyCmd)
}

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify <tag>",
	Short: "Verify the signature of a manifest",
	Long: `Verify the signature of a previously pulled or packed manifest in local storage.

At least one public key must be imported using 'kubectl mft key import' for verification.

Examples:
  # Verify a local manifest
  kubectl mft verify myapp:v1.0.0

  # Verify a manifest with registry reference
  kubectl mft verify registry.example.com/manifests/app:v1.0.0`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		verifyOpts.tag = args[0]
		return runVerify(cmd.Context())
	},
}

func runVerify(ctx context.Context) error {
	if !signature.PublicKeysExist() {
		return fmt.Errorf("no verification keys found, run 'kubectl mft key import <file>' to import a public key")
	}

	r, err := oci.NewRepository(verifyOpts.tag)
	if err != nil {
		return err
	}

	verifier, err := signature.NewVerifierFromKeyDir()
	if err != nil {
		return err
	}

	err = verifier.Verify(ctx, r.LayoutPath(), r.Tag())
	if err != nil {
		return err
	}

	fmt.Printf("Verified %s: signature is valid\n", r.Tag())
	return nil
}
