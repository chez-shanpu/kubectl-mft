/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/repository"
)

const (
	dumpOutputFlag      = "output"
	dumpOutputShortFlag = "o"
	dumpTagFlag         = "tag"
	dumpTagShortFlag    = "t"
)

type DumpOpts struct {
	output string
	tag    string
}

var dumpOpts DumpOpts

func init() {
	rootCmd.AddCommand(dumpCmd)

	flag := dumpCmd.Flags()
	flag.StringVarP(&dumpOpts.output, dumpOutputFlag, dumpOutputShortFlag, "", "Output file path (default: stdout)")
	flag.StringVarP(&dumpOpts.tag, dumpTagFlag, dumpTagShortFlag, "", "OCI reference for the manifest to dump (e.g., registry.example.com/repo:tag)")

	_ = dumpCmd.MarkFlagRequired(dumpTagFlag)
}

// dumpCmd represents the dump command
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump a manifest from local OCI layout storage",
	Long: `Dump retrieves and outputs a Kubernetes manifest from local OCI layout storage.

This command reads a previously packed manifest from the local OCI layout and outputs
its contents either to stdout or to a specified file. The manifest must have been
previously packed using the 'pack' command.

Examples:
  # Dump manifest to stdout
  kubectl mft dump -t registry.example.com/manifests/app:v1.0.0

  # Dump manifest to a file
  kubectl mft dump -t localhost/myapp:latest -o restored-manifest.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDump(cmd.Context())
	},
}

func runDump(ctx context.Context) error {
	r := repository.NewRepository(dumpOpts.tag)
	return mft.Dump(ctx, r, dumpOpts.output)
}
