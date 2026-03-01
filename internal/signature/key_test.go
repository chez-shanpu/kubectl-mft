// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package signature

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func setupTestKeyDir(t *testing.T) (cleanup func()) {
	t.Helper()
	origKeyDir := keyDir
	tmpDir := t.TempDir()
	keyDir = tmpDir
	return func() {
		keyDir = origKeyDir
	}
}

func TestGenerateKeyPair(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	// Generate a key pair
	if err := GenerateKeyPair("default", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Verify private key exists
	if !PrivateKeyExists("default") {
		t.Fatal("private key should exist after generation")
	}

	// Verify public key exists
	if !PublicKeysExist() {
		t.Fatal("public keys should exist after generation")
	}

	// Verify key files are on disk
	privPath := PrivateKeyPath("default")
	if _, err := os.Stat(privPath); err != nil {
		t.Fatalf("private key file not found: %v", err)
	}
	pubPath := filepath.Join(KeyDir(), "default.pub")
	if _, err := os.Stat(pubPath); err != nil {
		t.Fatalf("public key file not found: %v", err)
	}

	// Attempting to generate again without force should fail
	if err := GenerateKeyPair("default", false); err == nil {
		t.Fatal("GenerateKeyPair should fail when private key already exists")
	}

	// Generating with force should succeed
	if err := GenerateKeyPair("default", true); err != nil {
		t.Fatalf("GenerateKeyPair with force failed: %v", err)
	}
}

func TestGenerateKeyPairCustomName(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := GenerateKeyPair("mykey", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	privPath := filepath.Join(KeyDir(), "mykey.key")
	if _, err := os.Stat(privPath); err != nil {
		t.Fatalf("custom-named private key not found: %v", err)
	}

	pubPath := filepath.Join(KeyDir(), "mykey.pub")
	if _, err := os.Stat(pubPath); err != nil {
		t.Fatalf("custom-named public key not found: %v", err)
	}

	if !PrivateKeyExists("mykey") {
		t.Fatal("PrivateKeyExists should return true for custom-named key")
	}
}

func TestLoadPrivateKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := GenerateKeyPair("default", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	signer, err := LoadPrivateKey("default")
	if err != nil {
		t.Fatalf("LoadPrivateKey failed: %v", err)
	}

	ecdsaKey, ok := signer.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("loaded key should be an ECDSA private key")
	}
	if ecdsaKey.Curve != elliptic.P256() {
		t.Fatal("loaded key should use P-256 curve")
	}
}

func TestLoadAllPublicKeys(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := GenerateKeyPair("default", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	keys, err := LoadAllPublicKeys()
	if err != nil {
		t.Fatalf("LoadAllPublicKeys failed: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 public key, got %d", len(keys))
	}

	if _, ok := keys[0].(*ecdsa.PublicKey); !ok {
		t.Fatal("loaded key should be an ECDSA public key")
	}
}

func TestImportPublicKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	// Generate a test public key to import
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	// Write to temp file
	tmpFile := filepath.Join(t.TempDir(), "test.pub")
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	if err := os.WriteFile(tmpFile, pem.EncodeToMemory(block), 0o644); err != nil {
		t.Fatalf("failed to write test key: %v", err)
	}

	// Import
	if err := ImportPublicKey(tmpFile, "alice"); err != nil {
		t.Fatalf("ImportPublicKey failed: %v", err)
	}

	// Verify imported
	alicePath := filepath.Join(KeyDir(), "alice.pub")
	if _, err := os.Stat(alicePath); err != nil {
		t.Fatalf("imported public key not found: %v", err)
	}

	if !PublicKeysExist() {
		t.Fatal("public keys should exist after import")
	}
}

func TestImportPublicKeyDefaultName(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	// Generate a test public key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "bob.pub")
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	if err := os.WriteFile(tmpFile, pem.EncodeToMemory(block), 0o644); err != nil {
		t.Fatalf("failed to write test key: %v", err)
	}

	// Import with empty name (should use filename base)
	if err := ImportPublicKey(tmpFile, ""); err != nil {
		t.Fatalf("ImportPublicKey failed: %v", err)
	}

	bobPath := filepath.Join(KeyDir(), "bob.pub")
	if _, err := os.Stat(bobPath); err != nil {
		t.Fatalf("imported public key not found at expected path: %v", err)
	}
}

func TestImportInvalidPublicKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	// Write an invalid file
	tmpFile := filepath.Join(t.TempDir(), "invalid.pub")
	if err := os.WriteFile(tmpFile, []byte("not a valid key"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if err := ImportPublicKey(tmpFile, "invalid"); err == nil {
		t.Fatal("ImportPublicKey should fail with invalid key data")
	}
}

func TestDeletePublicKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := GenerateKeyPair("default", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if err := DeletePublicKey("default"); err != nil {
		t.Fatalf("DeletePublicKey failed: %v", err)
	}

	pubPath := filepath.Join(KeyDir(), "default.pub")
	if _, err := os.Stat(pubPath); !os.IsNotExist(err) {
		t.Fatal("public key should be deleted")
	}
}

func TestDeletePrivateKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := GenerateKeyPair("default", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if err := DeletePrivateKey("default"); err != nil {
		t.Fatalf("DeletePrivateKey failed: %v", err)
	}

	privPath := PrivateKeyPath("default")
	if _, err := os.Stat(privPath); !os.IsNotExist(err) {
		t.Fatal("private key should be deleted")
	}
}

func TestDeleteNonexistentPrivateKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := DeletePrivateKey("nonexistent"); err == nil {
		t.Fatal("DeletePrivateKey should fail for nonexistent key")
	}
}

func TestDeleteNonexistentPublicKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := DeletePublicKey("nonexistent"); err == nil {
		t.Fatal("DeletePublicKey should fail for nonexistent key")
	}
}

func TestListKeys(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := GenerateKeyPair("default", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	keys, err := ListKeys()
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}

	// Should have default.key + default.pub
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	hasPrivate := false
	hasPublic := false
	for _, k := range keys {
		if k.Type == "private" && k.Name == "default" {
			hasPrivate = true
		}
		if k.Type == "public" && k.Name == "default" {
			hasPublic = true
		}
	}
	if !hasPrivate {
		t.Fatal("should list private key")
	}
	if !hasPublic {
		t.Fatal("should list default public key")
	}
}

func TestListKeysEmpty(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	keys, err := ListKeys()
	if err != nil {
		t.Fatalf("ListKeys failed: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected 0 keys in empty directory, got %d", len(keys))
	}
}

func TestExportPublicKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if err := GenerateKeyPair("default", false); err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	data, err := ExportPublicKey("default")
	if err != nil {
		t.Fatalf("ExportPublicKey failed: %v", err)
	}

	// Validate it's a valid PEM public key
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("exported key should be valid PEM")
	}
	if block.Type != "PUBLIC KEY" {
		t.Fatalf("expected PUBLIC KEY PEM block, got %s", block.Type)
	}
}

func TestExportNonexistentPublicKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if _, err := ExportPublicKey("nonexistent"); err == nil {
		t.Fatal("ExportPublicKey should fail for nonexistent key")
	}
}

func TestPrivateKeyExistsWhenNoKey(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if PrivateKeyExists("default") {
		t.Fatal("private key should not exist in empty directory")
	}
}

func TestValidateKeyNameRejectsPathTraversal(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	cases := []struct {
		name    string
		keyName string
	}{
		{"dot-dot slash", "../evil"},
		{"dot-dot only", ".."},
		{"forward slash", "sub/key"},
		{"backslash", "sub\\key"},
		{"absolute path", "/etc/passwd"},
		{"empty name for generate", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := GenerateKeyPair(tc.keyName, false); err == nil && tc.keyName != "" {
				t.Fatalf("GenerateKeyPair should reject key name %q", tc.keyName)
			}
			if err := DeletePrivateKey(tc.keyName); err == nil {
				t.Fatalf("DeletePrivateKey should reject key name %q", tc.keyName)
			}
			if err := DeletePublicKey(tc.keyName); err == nil {
				t.Fatalf("DeletePublicKey should reject key name %q", tc.keyName)
			}
			if _, err := ExportPublicKey(tc.keyName); err == nil && tc.keyName != "" {
				t.Fatalf("ExportPublicKey should reject key name %q", tc.keyName)
			}
			if _, err := LoadPrivateKey(tc.keyName); err == nil {
				t.Fatalf("LoadPrivateKey should reject key name %q", tc.keyName)
			}
			if PrivateKeyExists(tc.keyName) {
				t.Fatalf("PrivateKeyExists should return false for key name %q", tc.keyName)
			}
		})
	}
}

func TestPublicKeysExistWhenNoKeys(t *testing.T) {
	cleanup := setupTestKeyDir(t)
	defer cleanup()

	if PublicKeysExist() {
		t.Fatal("public keys should not exist in empty directory")
	}
}
