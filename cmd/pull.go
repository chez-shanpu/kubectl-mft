// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
)

const (
	pullTagFlag      = "tag"
	pullTagShortFlag = "t"
)

func init() {
	rootCmd.AddCommand(pullCmd)

	flag := pullCmd.Flags()
	flag.StringP(pullTagFlag, pullTagShortFlag, "", "OCI reference for the manifest to pull (e.g., registry.example.com/repo:tag)")

	_ = pullCmd.MarkFlagRequired(pullTagFlag)
}

type pullOpts struct {
	tag string
}

func (p *pullOpts) parse(f *pflag.FlagSet) {
	p.tag = f.Lookup(pullTagFlag).Value.String()
}

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull a manifest from an OCI registry",
	Long: `Pull downloads a previously pushed Kubernetes manifest from an OCI-compliant registry
to local storage for further use.

The manifest must have been previously pushed to the registry using the 'push' command.
Authentication is handled through Docker credential store, so ensure you are logged
into the source registry using 'docker login' before pulling.

Examples:
  # Pull manifest from Docker Hub
  kubectl mft pull -t docker.io/myuser/my-app:v1.0.0
  
  # Pull from a private registry
  kubectl mft pull -t registry.company.com/team/app:latest
  
  # Pull from localhost registry
  kubectl mft pull -t localhost:5000/test-app:dev`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opt := &pullOpts{}
		opt.parse(cmd.Flags())
		return runPull(opt)
	},
}

func runPull(opt *pullOpts) error {
	ctx := context.Background()
	return mft.Pull(ctx, opt.tag)
}
