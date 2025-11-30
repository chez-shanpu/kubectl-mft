// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information. These are set via ldflags during build.
	version = "dev"
	commit  = "none"
)

const (
	OutputFlag      = "output"
	OutputShortFlag = "o"

	FileFlag      = "file"
	FileShortFlag = "f"

	ForceFlag      = "force"
	ForceShortFlag = "y"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "kubectl-mft",
	Short:        "A kubectl plugin for managing Kubernetes manifests",
	SilenceUsage: true,
	Version:      version,
}

func init() {
	// Customize version output template
	rootCmd.SetVersionTemplate(fmt.Sprintf("kubectl-mft version %s (commit: %s)\n", version, commit))
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
