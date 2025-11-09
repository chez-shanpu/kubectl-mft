// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
	"github.com/chez-shanpu/kubectl-mft/internal/repository"
)

const (
	listOutputFlag      = "output"
	listOutputShortFlag = "o"
)

type ListOpts struct {
	output string
}

var listOpts ListOpts

func init() {
	rootCmd.AddCommand(listCmd)

	flag := listCmd.Flags()
	flag.StringVarP(&listOpts.output, listOutputFlag, listOutputShortFlag, "table", "Output format (table, json, yaml)")
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all manifests in local OCI layout storage",
	Long: `List retrieves and displays all Kubernetes manifests stored in local OCI layout.

This command scans the local OCI layout directory and shows information about all
stored manifests including their repository names, tags, sizes, and creation times.

Output formats:
  - table: Human-readable table format (default)
  - json:  JSON format
  - yaml:  YAML format

Examples:
  # List all manifests in table format
  kubectl mft list

  # List in JSON format
  kubectl mft list -o json

  # List in YAML format
  kubectl mft list --output yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList(cmd.Context())
	},
}

func runList(ctx context.Context) error {
	r := repository.NewRepository("")
	infos, err := mft.List(ctx, r)
	if err != nil {
		return err
	}

	switch listOpts.output {
	case "table":
		return printTable(infos)
	case "json":
		return printJSON(infos)
	case "yaml":
		return printYAML(infos)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: table, json, yaml)", listOpts.output)
	}
}

func printTable(infos []mft.Info) error {
	if len(infos) == 0 {
		fmt.Println("No manifests found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tTAG\tSIZE\tCREATED")

	for _, info := range infos {
		size := formatSize(info.Size)
		created := info.Created.Format("2006-01-02 15:04:05")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Repository, info.Tag, size, created)
	}

	return w.Flush()
}

func printJSON(infos []mft.Info) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(infos)
}

func printYAML(infos []mft.Info) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(infos)
}

// formatSize formats byte size to human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
