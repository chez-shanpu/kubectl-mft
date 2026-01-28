// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/oci"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
)

type Registry struct{}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) List(ctx context.Context) (*mft.ListResult, error) {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return mft.NewListResult(nil), nil
	}

	var info []*mft.Info
	if err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() || path == baseDir {
			return nil
		}

		// Check if this directory contains an index.json (OCI layout marker)
		if _, err := os.Stat(filepath.Join(path, "index.json")); err != nil {
			// not an OCI layout directory
			return nil
		}
		i, err := readIndex(path)
		if err != nil {
			return fmt.Errorf("warning: failed to read OCI index at %s: %w", path, err)
		}
		info = append(info, i...)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to walk manifest directory: %w", err)
	}

	return mft.NewListResult(info), nil
}

// readIndex reads the index.json file and extracts manifest information
func readIndex(indexDir string) ([]*mft.Info, error) {
	repoName, err := getRepoName(indexDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository name: %w", err)
	}

	if _, err := oci.New(indexDir); err != nil {
		return nil, fmt.Errorf("failed to open OCI store: %w", err)
	}

	indexData, err := os.ReadFile(filepath.Join(indexDir, "index.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read index.json: %w", err)
	}

	var index *v1.Index
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index.json: %w", err)
	}

	var infos []*mft.Info
	for _, manifest := range index.Manifests {
		tag := manifest.Annotations["org.opencontainers.image.ref.name"]
		if tag == "" {
			continue // Skip manifests without tags
		}

		// Get the creation time from the manifest blob file
		created, size, err := getManifestMetadata(indexDir, manifest.Digest)
		if err != nil {
			return nil, fmt.Errorf("warning: failed to get metadata for %s/%s: %w", repoName, tag, err)
		}

		infos = append(infos, &mft.Info{
			Repository: repoName,
			Tag:        tag,
			Size:       formatSize(size),
			Created:    created,
		})
	}

	return infos, nil
}

func getRepoName(indexDir string) (string, error) {
	relPath, err := filepath.Rel(baseDir, indexDir)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	repoName := filepath.ToSlash(relPath)

	// Strip the default registry prefix for display
	repoName = strings.TrimPrefix(repoName, DefaultRegistry+"/")

	return repoName, nil
}

// getManifestMetadata gets the creation time and size of a manifest blob
func getManifestMetadata(indexDir string, digest digest.Digest) (created time.Time, size int64, err error) {
	// Construct a blob path
	blobDir := filepath.Join(indexDir, "blobs", digest.Algorithm().String(), digest.Encoded())

	// Get file info
	fileInfo, err := os.Stat(blobDir)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("failed to stat blob file: %w", err)
	}

	return fileInfo.ModTime(), fileInfo.Size(), nil
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
