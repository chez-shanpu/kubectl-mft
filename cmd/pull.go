// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

type PullOpts struct {
	tag        string
	skipVerify bool
}

var pullOpts PullOpts

func init() {
	rootCmd.AddCommand(pullCmd)

	flag := pullCmd.Flags()
	flag.BoolVar(&pullOpts.skipVerify, "skip-verify", false, "Skip signature verification after pulling")
}

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull <tag>",
	Short: "Pull a manifest from an OCI registry",
	Long: `Pull downloads a previously pushed Kubernetes manifest from an OCI-compliant registry
to local storage for further use.

The manifest must have been previously pushed to the registry using the 'push' command.
Authentication is handled through Docker credential store, so ensure you are logged
into the source registry using 'docker login' before pulling.

Examples:
  # Pull manifest from Docker Hub
  kubectl mft pull docker.io/myuser/my-app:v1.0.0

  # Pull from a private registry
  kubectl mft pull registry.company.com/team/app:latest

  # Pull from localhost registry
  kubectl mft pull localhost:5000/test-app:dev`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pullOpts.tag = args[0]
		return runPull(cmd.Context())
	},
}

func runPull(ctx context.Context) error {
	r, err := oci.NewRepository(pullOpts.tag)
	if err != nil {
		return err
	}

	// Check if manifest already exists locally before pull
	existedBefore, err := r.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check local manifest: %w", err)
	}

	if err := mft.Pull(ctx, r); err != nil {
		return err
	}

	if !pullOpts.skipVerify {
		if !signature.PublicKeysExist() {
			return handleVerifyFailure(ctx, r, existedBefore, fmt.Errorf("no verification keys found, run 'kubectl mft key import <file>' to import a public key, or use '--skip-verify' to skip verification"))
		}
		verifier, err := signature.NewVerifierFromKeyDir()
		if err != nil {
			return handleVerifyFailure(ctx, r, existedBefore, err)
		}
		if err := verifier.Verify(ctx, r.LayoutPath(), r.Tag()); err != nil {
			return handleVerifyFailure(ctx, r, existedBefore, fmt.Errorf("signature verification failed: %w", err))
		}
	}

	return nil
}

func handleVerifyFailure(ctx context.Context, r *oci.Repository, existedBefore bool, originalErr error) error {
	if existedBefore {
		// Manifest existed before pull; don't attempt further deletion
		return originalErr
	}
	return deletePulledData(ctx, r, originalErr)
}

func deletePulledData(ctx context.Context, r *oci.Repository, originalErr error) error {
	if _, deleteErr := mft.Delete(ctx, r); deleteErr != nil {
		return errors.Join(originalErr, fmt.Errorf("failed to clean up pulled data: %w", deleteErr))
	}
	return originalErr
}
