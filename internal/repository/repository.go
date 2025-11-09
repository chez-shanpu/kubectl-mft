// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"

	"github.com/chez-shanpu/kubectl-mft/internal/mft"
)

type Repository struct {
	tag string
}

func NewRepository(tag string) *Repository {
	return &Repository{tag: tag}
}

func (r *Repository) Dump(ctx context.Context) ([]byte, error) {
	ref, err := parseReference(r.tag)
	if err != nil {
		return nil, err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return nil, err
	}

	desc, err := layoutStore.Resolve(ctx, ref.ReferenceOrDefault())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve reference %store: %w", ref.ReferenceOrDefault(), err)
	}

	manifestJSON, err := content.FetchAll(ctx, layoutStore, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content for %s: %w", ref.ReferenceOrDefault(), err)
	}

	var m v1.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	if len(m.Layers) != 1 {
		return nil, fmt.Errorf("expected a single layer in the manifest, got %d", len(m.Layers))
	}

	return content.FetchAll(ctx, layoutStore, m.Layers[0])
}

func (r *Repository) Path(ctx context.Context) (string, error) {
	ref, err := parseReference(r.tag)
	if err != nil {
		return "", err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return "", err
	}

	desc, err := layoutStore.Resolve(ctx, ref.ReferenceOrDefault())
	if err != nil {
		return "", fmt.Errorf("failed to resolve reference %s: %w", ref.ReferenceOrDefault(), err)
	}

	manifestJSON, err := content.FetchAll(ctx, layoutStore, desc)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content for %s: %w", ref.ReferenceOrDefault(), err)
	}

	var m v1.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return "", fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	if len(m.Layers) != 1 {
		return "", fmt.Errorf("expected a single layer in the manifest, got %d", len(m.Layers))
	}

	layerDigest := m.Layers[0].Digest
	blobPath := filepath.Join(baseDir, repoName(ref), "blobs", layerDigest.Algorithm().String(), layerDigest.Encoded())

	return blobPath, nil
}

func (r *Repository) Save(ctx context.Context, manifestPath string) error {
	ref, err := parseReference(r.tag)
	if err != nil {
		return err
	}

	fs, err := newFileStore(ctx, ref, manifestPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fs.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close manifestPath content: %v\n", closeErr)
		}
	}()

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return err
	}

	return copyRepo(ctx, fs, layoutStore, ref)
}

func (r *Repository) Push(ctx context.Context) error {
	ref, err := parseReference(r.tag)
	if err != nil {
		return err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return err
	}

	repo, err := newAuthenticatedRepository(ref)
	if err != nil {
		return err
	}

	return copyRepo(ctx, layoutStore, repo, ref)
}

func (r *Repository) Pull(ctx context.Context) error {
	ref, err := parseReference(r.tag)
	if err != nil {
		return err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return err
	}

	repo, err := newAuthenticatedRepository(ref)
	if err != nil {
		return err
	}

	return copyRepo(ctx, repo, layoutStore, ref)
}

func (r *Repository) List(ctx context.Context) ([]mft.Info, error) {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return []mft.Info{}, nil
	}

	var infos []mft.Info

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() || path == baseDir {
			return nil
		}

		// Check if this directory contains an index.json (OCI layout marker)
		indexPath := filepath.Join(path, "index.json")
		if _, err := os.Stat(indexPath); err == nil {
			repoInfos, err := readOCIIndex(ctx, path, indexPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to read OCI index at %s: %v\n", indexPath, err)
				return nil
			}
			infos = append(infos, repoInfos...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk manifest directory: %w", err)
	}

	// Sort by repository name, then by tag
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Repository != infos[j].Repository {
			return infos[i].Repository < infos[j].Repository
		}
		return infos[i].Tag < infos[j].Tag
	})

	return infos, nil
}

// readOCIIndex reads the index.json file and extracts manifest information
func readOCIIndex(ctx context.Context, layoutPath, indexPath string) ([]mft.Info, error) {
	relPath, err := filepath.Rel(baseDir, layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}
	repoName := filepath.ToSlash(relPath)

	_, err = oci.New(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open OCI store: %w", err)
	}

	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index.json: %w", err)
	}

	var index struct {
		Manifests []struct {
			Digest      string            `json:"digest"`
			Size        int64             `json:"size"`
			Annotations map[string]string `json:"annotations"`
		} `json:"manifests"`
	}

	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index.json: %w", err)
	}

	var infos []mft.Info
	for _, manifest := range index.Manifests {
		tag := manifest.Annotations["org.opencontainers.image.ref.name"]
		if tag == "" {
			continue // Skip manifests without tags
		}

		// Get the creation time from the manifest blob file
		created, size, err := getManifestMetadata(layoutPath, manifest.Digest)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to get metadata for %s:%s: %v\n", repoName, tag, err)
			continue
		}

		infos = append(infos, mft.Info{
			Repository: repoName,
			Tag:        tag,
			Size:       formatSize(size),
			Created:    created,
		})
	}

	return infos, nil
}

// getManifestMetadata gets the creation time and size of a manifest blob
func getManifestMetadata(layoutPath, digest string) (created time.Time, size int64, err error) {
	// Parse digest (format: "sha256:abc123...")
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return time.Time{}, 0, fmt.Errorf("invalid digest format: %s", digest)
	}
	algorithm := parts[0]
	encoded := parts[1]

	// Construct blob path
	blobPath := filepath.Join(layoutPath, "blobs", algorithm, encoded)

	// Get file info
	fileInfo, err := os.Stat(blobPath)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("failed to stat blob file: %w", err)
	}

	return fileInfo.ModTime(), fileInfo.Size(), nil
}

func (r *Repository) Delete(ctx context.Context) (*mft.DeleteResult, error) {
	ref, err := parseReference(r.tag)
	if err != nil {
		return nil, err
	}

	layoutStore, err := newOCILayoutStore(ref)
	if err != nil {
		return nil, err
	}

	desc, err := layoutStore.Resolve(ctx, ref.ReferenceOrDefault())
	if err != nil {
		// If not found, return nil (idempotent behavior)
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve reference %s: %w", ref.ReferenceOrDefault(), err)
	}

	manifestJSON, err := content.FetchAll(ctx, layoutStore, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	var m v1.Manifest
	if err := json.Unmarshal(manifestJSON, &m); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	totalSize := desc.Size
	for _, layer := range m.Layers {
		totalSize += layer.Size
	}

	layoutPath := filepath.Join(baseDir, repoName(ref))
	indexPath := filepath.Join(layoutPath, "index.json")

	if err := removeTagFromIndex(indexPath, ref.ReferenceOrDefault()); err != nil {
		return nil, fmt.Errorf("failed to remove tag from index: %w", err)
	}

	referencedBlobs, err := getReferencedBlobs(ctx, layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get referenced blobs: %w", err)
	}

	// Find orphaned blobs (blobs only referenced by deleted manifest)
	var orphanedBlobs []string
	for _, layer := range m.Layers {
		digest := layer.Digest.String()
		if !referencedBlobs[digest] {
			orphanedBlobs = append(orphanedBlobs, digest)
		}
	}
	// Also check manifest blob itself
	manifestDigest := desc.Digest.String()
	if !referencedBlobs[manifestDigest] {
		orphanedBlobs = append(orphanedBlobs, manifestDigest)
	}

	for _, digest := range orphanedBlobs {
		if err := deleteBlobFile(layoutPath, digest); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete blob %s: %v\n", digest, err)
		}
	}

	// Check if repository directory should be removed
	indexData, err := os.ReadFile(indexPath)
	if err == nil {
		var index struct {
			Manifests []interface{} `json:"manifests"`
		}
		if err := json.Unmarshal(indexData, &index); err == nil {
			if len(index.Manifests) == 0 {
				// Remove entire repository directory if no manifests remain
				if err := os.RemoveAll(layoutPath); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to remove repository directory: %v\n", err)
				}
			}
		}
	}

	return &mft.DeleteResult{
		Repository:   repoName(ref),
		Tag:          ref.Reference,
		Size:         formatSize(totalSize),
		RemovedBlobs: len(orphanedBlobs),
	}, nil
}

// removeTagFromIndex removes a tag entry from index.json
func removeTagFromIndex(indexPath, tag string) error {
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("failed to read index.json: %w", err)
	}

	var index struct {
		SchemaVersion int `json:"schemaVersion"`
		Manifests     []struct {
			MediaType   string            `json:"mediaType"`
			Digest      string            `json:"digest"`
			Size        int64             `json:"size"`
			Annotations map[string]string `json:"annotations"`
		} `json:"manifests"`
	}

	if err := json.Unmarshal(indexData, &index); err != nil {
		return fmt.Errorf("failed to unmarshal index.json: %w", err)
	}

	var newManifests []struct {
		MediaType   string            `json:"mediaType"`
		Digest      string            `json:"digest"`
		Size        int64             `json:"size"`
		Annotations map[string]string `json:"annotations"`
	}

	for _, manifest := range index.Manifests {
		if manifest.Annotations["org.opencontainers.image.ref.name"] != tag {
			newManifests = append(newManifests, manifest)
		}
	}

	index.Manifests = newManifests

	// Write back updated index
	updatedData, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index.json: %w", err)
	}

	if err := os.WriteFile(indexPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write index.json: %w", err)
	}

	return nil
}

// getReferencedBlobs scans all manifests in index.json and returns referenced blob digests
func getReferencedBlobs(ctx context.Context, layoutPath string) (map[string]bool, error) {
	indexPath := filepath.Join(layoutPath, "index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read index.json: %w", err)
	}

	var index struct {
		Manifests []struct {
			Digest string `json:"digest"`
		} `json:"manifests"`
	}

	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index.json: %w", err)
	}

	referenced := make(map[string]bool)

	for _, manifestEntry := range index.Manifests {
		referenced[manifestEntry.Digest] = true

		parts := strings.SplitN(manifestEntry.Digest, ":", 2)
		if len(parts) != 2 {
			continue
		}
		algorithm := parts[0]
		encoded := parts[1]

		manifestBlobPath := filepath.Join(layoutPath, "blobs", algorithm, encoded)
		manifestData, err := os.ReadFile(manifestBlobPath)
		if err != nil {
			continue
		}

		var manifest v1.Manifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			continue
		}

		for _, layer := range manifest.Layers {
			referenced[layer.Digest.String()] = true
		}

		if manifest.Config.Digest.String() != "" {
			referenced[manifest.Config.Digest.String()] = true
		}
	}

	return referenced, nil
}

// deleteBlobFile removes a blob file by digest
func deleteBlobFile(layoutPath, digest string) error {
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid digest format: %s", digest)
	}
	algorithm := parts[0]
	encoded := parts[1]

	blobPath := filepath.Join(layoutPath, "blobs", algorithm, encoded)
	if err := os.Remove(blobPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove blob file: %w", err)
	}

	return nil
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
