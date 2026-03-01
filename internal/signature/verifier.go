// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package signature

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/oci"
)

// Verifier performs verification on local OCI layouts.
type Verifier struct {
	publicKeys []crypto.PublicKey
}

// NewVerifier creates a new Verifier with the given public keys.
func NewVerifier(publicKeys []crypto.PublicKey) *Verifier {
	return &Verifier{
		publicKeys: publicKeys,
	}
}

// NewVerifierFromKeyDir creates a Verifier by loading all public keys from the key directory.
func NewVerifierFromKeyDir() (*Verifier, error) {
	pubKeys, err := LoadAllPublicKeys()
	if err != nil {
		return nil, err
	}
	return NewVerifier(pubKeys), nil
}

// Verify verifies the manifest identified by tag in the OCI layout at layoutPath.
func (v *Verifier) Verify(ctx context.Context, layoutPath, tag string) error {
	if len(v.publicKeys) == 0 {
		return fmt.Errorf("no public keys available for verification")
	}

	store, err := oci.New(layoutPath)
	if err != nil {
		return fmt.Errorf("failed to open OCI layout: %w", err)
	}

	// Resolve the manifest descriptor
	desc, err := store.Resolve(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to resolve tag %q: %w", tag, err)
	}

	// Find signature artifacts via predecessors (referrers)
	predecessors, err := store.Predecessors(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to get predecessors: %w", err)
	}

	// Try to verify with any signature and any public key
	var extractErrs []string
	foundSignature := false
	for _, p := range predecessors {
		sig, isSignature, err := tryExtractSignature(ctx, store, p)
		if !isSignature {
			continue
		}
		foundSignature = true
		if err != nil {
			extractErrs = append(extractErrs, err.Error())
			continue
		}

		for _, pubKey := range v.publicKeys {
			if verifySignature(pubKey, desc.Digest, sig) {
				return nil
			}
		}
	}

	if !foundSignature {
		return fmt.Errorf("no signature found for %q", tag)
	}

	msg := fmt.Sprintf("signature verification failed for %q: none of the available public keys could verify the signature", tag)
	if len(extractErrs) > 0 {
		msg += fmt.Sprintf("; additionally, %d signature(s) could not be read: %s", len(extractErrs), strings.Join(extractErrs, "; "))
	}
	return errors.New(msg)
}

// verifySignature verifies an ECDSA signature against a digest.
func verifySignature(pubKey crypto.PublicKey, d digest.Digest, sig []byte) bool {
	ecdsaKey, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return false
	}
	hash := sha256.Sum256([]byte(d.String()))
	return ecdsa.VerifyASN1(ecdsaKey, hash[:], sig)
}

// tryExtractSignature attempts to extract a signature from a predecessor descriptor.
// Returns (signature, true, nil) if the descriptor is a signature artifact and extraction succeeded.
// Returns (nil, true, err) if it's a signature artifact but extraction failed.
// Returns (nil, false, nil) if the descriptor is not a signature artifact.
func tryExtractSignature(ctx context.Context, store *oci.Store, desc v1.Descriptor) ([]byte, bool, error) {
	isSignature := desc.ArtifactType == SignatureArtifactType

	if !isSignature && desc.MediaType != v1.MediaTypeImageManifest {
		return nil, false, nil
	}

	// Fetch the manifest
	rc, err := store.Fetch(ctx, desc)
	if err != nil {
		if isSignature {
			return nil, true, fmt.Errorf("failed to fetch signature manifest: %w", err)
		}
		return nil, false, nil
	}
	defer rc.Close()

	manifestBytes, err := io.ReadAll(rc)
	if err != nil {
		if isSignature {
			return nil, true, fmt.Errorf("failed to read signature manifest: %w", err)
		}
		return nil, false, nil
	}

	var manifest v1.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		if isSignature {
			return nil, true, fmt.Errorf("failed to unmarshal signature manifest: %w", err)
		}
		return nil, false, nil
	}

	// If ArtifactType wasn't in the descriptor, check the manifest body
	if !isSignature {
		if manifest.ArtifactType != SignatureArtifactType {
			return nil, false, nil
		}
		isSignature = true
	}

	if len(manifest.Layers) == 0 {
		return nil, true, fmt.Errorf("signature manifest has no layers")
	}

	// Fetch the signature blob
	sigRC, err := store.Fetch(ctx, manifest.Layers[0])
	if err != nil {
		return nil, true, fmt.Errorf("failed to fetch signature blob: %w", err)
	}
	defer sigRC.Close()

	sig, err := io.ReadAll(sigRC)
	if err != nil {
		return nil, true, err
	}
	return sig, true, nil
}
