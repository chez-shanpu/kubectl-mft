// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package signature

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
)

const (
	// SignatureArtifactType is the artifact type for kubectl-mft signatures.
	SignatureArtifactType = "application/vnd.kubectl-mft.signature.v1"

	// SignatureMediaType is the media type for the signature layer.
	SignatureMediaType = "application/vnd.kubectl-mft.signature.v1+der"
)

// SignResult holds the result of a signing operation.
type SignResult struct {
	Digest string
}

// Signer performs signing on local OCI layouts.
type Signer struct {
	privateKey crypto.Signer
}

// NewSigner creates a new Signer with the given private key.
func NewSigner(privateKey crypto.Signer) *Signer {
	return &Signer{
		privateKey: privateKey,
	}
}

// NewSignerFromKeyDir creates a Signer by loading a private key from the key directory.
func NewSignerFromKeyDir(keyName string) (*Signer, error) {
	privKey, err := LoadPrivateKey(keyName)
	if err != nil {
		return nil, err
	}
	return NewSigner(privKey), nil
}

// Sign signs the manifest identified by tag in the OCI layout at layoutPath.
func (s *Signer) Sign(ctx context.Context, layoutPath, tag string) (*SignResult, error) {
	if s.privateKey == nil {
		return nil, fmt.Errorf("no private key available for signing")
	}

	store, err := oci.New(layoutPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open OCI layout: %w", err)
	}

	// Resolve the manifest descriptor
	desc, err := store.Resolve(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tag %q: %w", tag, err)
	}

	// Sign the manifest digest
	sig, err := signDigest(s.privateKey, desc.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to sign manifest: %w", err)
	}

	// Push signature blob to the store
	sigDigest := digest.FromBytes(sig)
	sigDesc := v1.Descriptor{
		MediaType: SignatureMediaType,
		Digest:    sigDigest,
		Size:      int64(len(sig)),
	}

	if err := store.Push(ctx, sigDesc, bytes.NewReader(sig)); err != nil {
		return nil, fmt.Errorf("failed to push signature blob: %w", err)
	}

	// Pack a manifest with the subject pointing to the signed manifest
	sigManifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, SignatureArtifactType, oras.PackManifestOptions{
		Subject: &desc,
		Layers:  []v1.Descriptor{sigDesc},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pack signature manifest: %w", err)
	}

	return &SignResult{
		Digest: sigManifestDesc.Digest.String(),
	}, nil
}

// signDigest signs the given digest using ECDSA with SHA-256.
func signDigest(key crypto.Signer, d digest.Digest) ([]byte, error) {
	hash := sha256.Sum256([]byte(d.String()))
	return key.Sign(rand.Reader, hash[:], crypto.SHA256)
}
