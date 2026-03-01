// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package signature

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
)

// setupTestOCILayout creates a temporary OCI layout with a test manifest and returns the layout path and tag.
func setupTestOCILayout(t *testing.T) (string, string) {
	t.Helper()

	layoutPath := t.TempDir()
	tag := "v1.0.0"

	store, err := oci.New(layoutPath)
	if err != nil {
		t.Fatalf("failed to create OCI store: %v", err)
	}

	ctx := context.Background()

	// Create a test content blob
	content := []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test\n")
	contentDesc := v1.Descriptor{
		MediaType: "application/vnd.kubectl-mft.content.v1+yaml",
		Digest:    digest.FromBytes(content),
		Size:      int64(len(content)),
	}

	if err := store.Push(ctx, contentDesc, bytes.NewReader(content)); err != nil {
		t.Fatalf("failed to push content: %v", err)
	}

	// Pack a manifest
	manifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "application/vnd.kubectl-mft.v1", oras.PackManifestOptions{
		Layers: []v1.Descriptor{contentDesc},
	})
	if err != nil {
		t.Fatalf("failed to pack manifest: %v", err)
	}

	if err := store.Tag(ctx, manifestDesc, tag); err != nil {
		t.Fatalf("failed to tag manifest: %v", err)
	}

	return layoutPath, tag
}

func generateTestKeyPair(t *testing.T) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	return key, &key.PublicKey
}

func TestSignAndVerify(t *testing.T) {
	layoutPath, tag := setupTestOCILayout(t)
	privKey, pubKey := generateTestKeyPair(t)

	signer := NewSigner(privKey)
	verifier := NewVerifier([]crypto.PublicKey{pubKey})

	ctx := context.Background()

	// Sign
	signResult, err := signer.Sign(ctx, layoutPath, tag)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	if signResult.Digest == "" {
		t.Fatal("expected non-empty digest in sign result")
	}

	// Verify
	err = verifier.Verify(ctx, layoutPath, tag)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
}

func TestVerifyWithWrongKey(t *testing.T) {
	layoutPath, tag := setupTestOCILayout(t)
	privKey, _ := generateTestKeyPair(t)
	_, wrongPubKey := generateTestKeyPair(t) // Different key pair

	// Sign with one key
	signer := NewSigner(privKey)
	ctx := context.Background()

	if _, err := signer.Sign(ctx, layoutPath, tag); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify with wrong key
	verifier := NewVerifier([]crypto.PublicKey{wrongPubKey})
	err := verifier.Verify(ctx, layoutPath, tag)
	if err == nil {
		t.Fatal("Verify should fail with wrong public key")
	}
}

func TestVerifyNoSignature(t *testing.T) {
	layoutPath, tag := setupTestOCILayout(t)
	_, pubKey := generateTestKeyPair(t)

	verifier := NewVerifier([]crypto.PublicKey{pubKey})
	ctx := context.Background()

	err := verifier.Verify(ctx, layoutPath, tag)
	if err == nil {
		t.Fatal("Verify should fail when no signature exists")
	}
}

func TestSignWithoutPrivateKey(t *testing.T) {
	layoutPath, tag := setupTestOCILayout(t)

	signer := NewSigner(nil)
	ctx := context.Background()

	if _, err := signer.Sign(ctx, layoutPath, tag); err == nil {
		t.Fatal("Sign should fail without private key")
	}
}

func TestVerifyWithoutPublicKeys(t *testing.T) {
	layoutPath, tag := setupTestOCILayout(t)

	verifier := NewVerifier(nil)
	ctx := context.Background()

	if err := verifier.Verify(ctx, layoutPath, tag); err == nil {
		t.Fatal("Verify should fail without public keys")
	}
}

func TestVerifyMultiplePublicKeys(t *testing.T) {
	layoutPath, tag := setupTestOCILayout(t)
	privKey, correctPubKey := generateTestKeyPair(t)
	_, wrongPubKey := generateTestKeyPair(t)

	// Sign
	signer := NewSigner(privKey)
	ctx := context.Background()
	if _, err := signer.Sign(ctx, layoutPath, tag); err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Verify with multiple keys (wrong key first, correct key second)
	verifier := NewVerifier([]crypto.PublicKey{wrongPubKey, correctPubKey})
	err := verifier.Verify(ctx, layoutPath, tag)
	if err != nil {
		t.Fatalf("Verify should succeed when one of multiple keys matches: %v", err)
	}
}
