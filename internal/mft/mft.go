// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/goccy/go-yaml"
)

type Info struct {
	Repository string    `json:"repository" yaml:"repository"`
	Tag        string    `json:"tag" yaml:"tag"`
	Size       string    `json:"size" yaml:"size"`
	Created    time.Time `json:"created" yaml:"created"`
}

type Registry interface {
	List(ctx context.Context) (*ListResult, error)
}

type ListOutput string

const (
	ListTable ListOutput = "table"
	ListJson  ListOutput = "json"
	ListYaml  ListOutput = "yaml"
)

// ListResult represents information about a stored manifest
type ListResult struct {
	info []*Info
}

func NewListResult(info []*Info) *ListResult {
	return &ListResult{info: info}
}

func (r *ListResult) Print(output ListOutput) error {
	switch output {
	case ListTable:
		return r.printTable()
	case ListJson:
		return r.printJSON()
	case ListYaml:
		return r.printYAML()
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func (r *ListResult) Sort() {
	sort.Slice(r.info, func(i, j int) bool {
		if r.info[i].Repository != r.info[j].Repository {
			return r.info[i].Repository < r.info[j].Repository
		}
		return r.info[i].Tag < r.info[j].Tag
	})
}

func (r *ListResult) printTable() error {
	if len(r.info) == 0 {
		fmt.Println("No manifests found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "REPOSITORY\tTAG\tSIZE\tCREATED")

	for _, i := range r.info {
		created := i.Created.Format("2006-01-02 15:04:05")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", i.Repository, i.Tag, i.Size, created)
	}

	return w.Flush()
}

func (r *ListResult) printJSON() error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r.info)
}

func (r *ListResult) printYAML() error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(r.info)
}

func List(ctx context.Context, r Registry) (*ListResult, error) {
	return r.List(ctx)
}

type Repository interface {
	Copy(ctx context.Context, dest string) error
	Delete(ctx context.Context) (*DeleteResult, error)
	Dump(ctx context.Context) (*DumpResult, error)
	Path(ctx context.Context) (*PathResult, error)
	Pull(ctx context.Context) error
	Push(ctx context.Context) error
	Save(ctx context.Context, manifestPath string) error
}

// DeleteResult represents the result of a delete operation
type DeleteResult struct {
	repository string
	tag        string
}

func NewDeleteResult(repository string, tag string) *DeleteResult {
	return &DeleteResult{
		repository: repository,
		tag:        tag,
	}
}

func (r *DeleteResult) Print() {
	fmt.Printf("Deleted %s:%s\n", r.repository, r.tag)
}

type DumpResult struct {
	data []byte
}

func NewDumpResult(data []byte) *DumpResult {
	return &DumpResult{data: data}
}

func (r *DumpResult) Read(p []byte) (n int, err error) {
	return bytes.NewReader(r.data).Read(p)
}

func (r *DumpResult) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(r.data)
	return int64(n), err
}

type PathResult struct {
	path string
}

func NewPathResult(path string) *PathResult {
	return &PathResult{path: path}
}

func (r *PathResult) Print() {
	fmt.Println(r.path)
}

// Copy copies a manifest from the source repository to a new destination tag in local storage.
func Copy(ctx context.Context, r Repository, dest string) error {
	return r.Copy(ctx, dest)
}

// Delete removes a manifest from local OCI layout storage
func Delete(ctx context.Context, r Repository) (*DeleteResult, error) {
	return r.Delete(ctx)
}

// Dump retrieves and outputs a manifest from local OCI layout storage
func Dump(ctx context.Context, r Repository) (*DumpResult, error) {
	return r.Dump(ctx)
}

func Path(ctx context.Context, r Repository) (*PathResult, error) {
	return r.Path(ctx)
}

// Pull pulls a Kubernetes manifest from an OCI registry
func Pull(ctx context.Context, r Repository) error {
	return r.Pull(ctx)
}

// Push pushes a Kubernetes manifest to an OCI registry
func Push(ctx context.Context, r Repository) error {
	return r.Push(ctx)
}

// Save packages a Kubernetes manifest into OCI layout format
func Save(ctx context.Context, r Repository, manifest string) error {
	return r.Save(ctx, manifest)
}
