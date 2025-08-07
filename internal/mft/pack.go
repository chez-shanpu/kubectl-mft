package mft

import (
	"context"
	"fmt"
	"os"
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

// Pack packages a Kubernetes manifest into OCI layout format
func Pack(ctx context.Context, manifest string, tag string) error {
	ref, err := parseReference(tag)
	if err != nil {
		return err
	}

	manifestContent, err := prepareManifestContent(ctx, manifest, &ref)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := manifestContent.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close manifest content: %v\n", closeErr)
		}
	}()

	return createOCILayout(ctx, manifestContent, &ref)
}

// prepareManifestContent processes the manifest file and creates content descriptor
func prepareManifestContent(ctx context.Context, manifestPath string, ref *registry.Reference) (*ManifestContent, error) {
	workingDir := filepath.Join(workingDIR, manifestDIRName(ref))
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

// createOCILayout creates the final OCI layout store and copies the manifest
func createOCILayout(ctx context.Context, content *ManifestContent, ref *registry.Reference) error {
	layoutStore, err := createOCILayoutStore(ref)
	if err != nil {
		return err
	}

	if _, err := oras.Copy(ctx, content.FileStore, content.Tag, layoutStore, content.Tag, oras.DefaultCopyOptions); err != nil {
		return fmt.Errorf("failed to copy manifest: %w", err)
	}
	return nil
}
