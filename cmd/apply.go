// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
	"github.com/chez-shanpu/kubectl-mft/internal/signature"
)

type ApplyOpts struct {
	tag        string
	skipVerify bool
}

var applyOpts ApplyOpts

func init() {
	rootCmd.AddCommand(applyCmd)

	flag := applyCmd.Flags()
	flag.BoolVar(&applyOpts.skipVerify, "skip-verify", false, "Skip signature verification after pulling")
}

// applyCmd represents the apply command
var applyCmd = &cobra.Command{
	Use:   "apply <tag>",
	Short: "Apply a manifest to the current Kubernetes cluster",
	Long: `Apply downloads a manifest from an OCI-compliant registry (if not already present locally)
and applies it to the current Kubernetes cluster using 'kubectl apply'.

If the manifest is not found in local storage, it will be automatically pulled from the
remote registry before applying. Authentication is handled through Docker credential store,
so ensure you are logged into the source registry using 'docker login' if pulling from a
private registry.

Examples:
  # Apply a locally available manifest
  kubectl mft apply docker.io/myuser/my-app:v1.0.0

  # Apply from a remote registry (auto-pulls if not local)
  kubectl mft apply registry.company.com/team/app:latest

  # Apply without signature verification
  kubectl mft apply localhost:5000/test-app:dev --skip-verify`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		applyOpts.tag = args[0]
		return runApply(cmd.Context())
	},
}

func runApply(ctx context.Context) error {
	r, err := oci.NewRepository(applyOpts.tag)
	if err != nil {
		return err
	}

	exists, err := r.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check local manifest: %w", err)
	}

	if !exists {
		if err := mft.Pull(ctx, r); err != nil {
			return err
		}

		if !applyOpts.skipVerify {
			if !signature.PublicKeysExist() {
				return deletePulledData(ctx, r, fmt.Errorf("no verification keys found, run 'kubectl mft key import <file>' to import a public key, or use '--skip-verify' to skip verification"))
			}
			verifier, err := signature.NewVerifierFromKeyDir()
			if err != nil {
				return deletePulledData(ctx, r, err)
			}
			if err := verifier.Verify(ctx, r.LayoutPath(), r.Tag()); err != nil {
				return deletePulledData(ctx, r, fmt.Errorf("signature verification failed: %w", err))
			}
		}
	}

	res, err := mft.Dump(ctx, r)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, res); err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	kubectl := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	kubectl.Stdin = &buf
	kubectl.Stdout = os.Stdout
	kubectl.Stderr = os.Stderr

	if err := kubectl.Run(); err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	return nil
}
