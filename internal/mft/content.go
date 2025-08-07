// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry"
)

// ManifestContent holds processed manifest data
type ManifestContent struct {
	FileStore    *file.Store
	ContentDesc  v1.Descriptor
	ManifestDesc v1.Descriptor
	Tag          string
}

func (c *ManifestContent) Close() error {
	if c.FileStore != nil {
		return c.FileStore.Close()
	}
	return nil
}

// prepareManifestContent processes the manifest file and creates content descriptor
func prepareManifestContent(ctx context.Context, manifestPath string, ref *registry.Reference) (*ManifestContent, error) {
	workingDir := filepath.Join(workingDIR, manifestName(ref))
	fs, err := file.New(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %w", err)
	}

	path, err := filepath.Abs(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of %q: %w", manifestPath, err)
	}

	name := strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))
	contentDesc, err := fs.Add(ctx, name, contentMediaType, path)
	if err != nil {
		return nil, fmt.Errorf("failed to add content: %w", err)
	}

	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, artifactType, oras.PackManifestOptions{
		Layers: []v1.Descriptor{contentDesc},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pack manifest: %w", err)
	}

	// Tag the manifest
	tagRef := ref.ReferenceOrDefault()
	if err = fs.Tag(ctx, manifestDesc, tagRef); err != nil {
		return nil, fmt.Errorf("failed to tag manifest: %w", err)
	}

	return &ManifestContent{
		FileStore:    fs,
		ContentDesc:  contentDesc,
		ManifestDesc: manifestDesc,
		Tag:          tagRef,
	}, nil
}