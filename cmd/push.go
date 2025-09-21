// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/registry"
)

const (
	pushTagFlag      = "tag"
	pushTagShortFlag = "t"
)

type PushOpts struct {
	tag string
}

var pushOpts PushOpts

func init() {
	rootCmd.AddCommand(pushCmd)

	flag := pushCmd.Flags()
	flag.StringVarP(&pushOpts.tag, pushTagFlag, pushTagShortFlag, "", "OCI reference for the manifest to push (e.g., registry.example.com/repo:tag)")

	_ = pushCmd.MarkFlagRequired(pushTagFlag)
}

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push a packaged manifest to an OCI registry",
	Long: `Push uploads a previously packaged Kubernetes manifest to an OCI-compliant registry.

The manifest must be packaged using the 'pack' command before it can be pushed.
Authentication is handled through Docker credential store, so ensure you are logged
into the target registry using 'docker login' before pushing.

Examples:
  # Push manifest to Docker Hub
  kubectl mft push -t docker.io/myuser/my-app:v1.0.0
  
  # Push to a private registry
  kubectl mft push -t registry.company.com/team/app:latest
  
  # Push to localhost registry
  kubectl mft push -t localhost:5000/test-app:dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPush(cmd.Context())
	},
}

func runPush(ctx context.Context) error {
	r := registry.NewRegistry()
	return mft.Push(ctx, r, pushOpts.tag)
}
