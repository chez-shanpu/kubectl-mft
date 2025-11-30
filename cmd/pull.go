// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
)

type PullOpts struct {
	tag string
}

var pullOpts PullOpts

func init() {
	rootCmd.AddCommand(pullCmd)
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
	return mft.Pull(ctx, r)
}
