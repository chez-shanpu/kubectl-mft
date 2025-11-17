// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/oci"
)

type DumpOpts struct {
	output string
	tag    string
}

var dumpOpts DumpOpts

func init() {
	rootCmd.AddCommand(dumpCmd)

	flag := dumpCmd.Flags()
	flag.StringVarP(&dumpOpts.output, OutputFlag, OutputShortFlag, "", "Output file path (default: stdout)")
	flag.StringVarP(&dumpOpts.tag, TagFlag, TagShortFlag, "", "OCI reference for the manifest to dump (e.g., registry.example.com/repo:tag)")

	_ = dumpCmd.MarkFlagRequired(TagFlag)
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

func runDump(ctx context.Context) (err error) {
	r, err := oci.NewRepository(dumpOpts.tag)
	if err != nil {
		return err
	}

	res, err := mft.Dump(ctx, r)
	if err != nil {
		return err
	}

	var w io.Writer
	if dumpOpts.output == "" {
		w = os.Stdout
	} else {
		f, err := os.Create(dumpOpts.output)
		if err != nil {
			return err
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		w = f
		// show output file path after writing
		defer fmt.Println(dumpOpts.output)
	}

	_, err = io.Copy(w, res)
	return err
}
