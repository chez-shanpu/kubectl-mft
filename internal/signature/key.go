// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package signature

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	privKeyExt = ".key"
	pubKeyExt  = ".pub"
)

var keyDir string

// InitKeyDir initializes the key storage directory path.
// It checks the KUBECTL_MFT_KEY_DIR environment variable first,
// then falls back to the default location under the user's home directory.
func InitKeyDir() error {
	if dir := os.Getenv("KUBECTL_MFT_KEY_DIR"); dir != "" {
		keyDir = dir
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	keyDir = filepath.Join(home, ".local", "share", "kubectl-mft", "keys")
	return nil
}

// KeyInfo holds information about a stored key.
type KeyInfo struct {
	Name string
	Type string // "private" or "public"
	Path string
}

// KeyDir returns the key storage directory path.
func KeyDir() string {
	return keyDir
}

// validateKeyName checks that the key name is safe for use as a filename.
func validateKeyName(name string) error {
	if name == "" {
		return fmt.Errorf("key name must not be empty")
	}
	if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid key name %q: must not contain path separators or '..'", name)
	}
	if name != filepath.Base(name) {
		return fmt.Errorf("invalid key name %q: must be a simple filename", name)
	}
	return nil
}

// PrivateKeyPath returns the path to the named private key file.
func PrivateKeyPath(name string) string {
	return filepath.Join(keyDir, name+privKeyExt)
}

// PublicKeyPath returns the path to the named public key file.
func PublicKeyPath(name string) string {
	return filepath.Join(keyDir, name+pubKeyExt)
}

// PrivateKeyExists checks if a named private key exists in the key directory.
func PrivateKeyExists(name string) bool {
	if validateKeyName(name) != nil {
		return false
	}
	_, err := os.Stat(PrivateKeyPath(name))
	return err == nil
}

// PublicKeysExist checks if at least one public key exists in the key directory.
func PublicKeysExist() bool {
	entries, err := os.ReadDir(keyDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), pubKeyExt) {
			return true
		}
	}
	return false
}

// GenerateKeyPair generates an ECDSA P-256 key pair and stores it in the key directory.
// The private key is saved as <name>.key and the public key as <name>.pub.
// If name is empty, "default" is used.
func GenerateKeyPair(name string, force bool) error {
	if name == "" {
		name = "default"
	}

	if err := validateKeyName(name); err != nil {
		return err
	}

	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	privPath := PrivateKeyPath(name)
	if !force {
		if _, err := os.Stat(privPath); err == nil {
			return fmt.Errorf("private key already exists at %s (use --force to overwrite)", privPath)
		}
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ECDSA key: %w", err)
	}

	if err := writePrivateKey(privPath, key); err != nil {
		return err
	}

	pubPath := PublicKeyPath(name)
	if err := writePublicKey(pubPath, &key.PublicKey); err != nil {
		// Clean up the private key if public key write fails
		os.Remove(privPath)
		return err
	}

	return nil
}

// ImportPublicKey copies a PEM-encoded public key file into the key directory.
// If name is empty, the base name of srcPath (without extension) is used.
func ImportPublicKey(srcPath, name string) error {
	if name == "" {
		base := filepath.Base(srcPath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	if err := validateKeyName(name); err != nil {
		return err
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read public key file: %w", err)
	}

	// Validate that it's a valid PEM-encoded public key
	if _, err := parsePublicKeyPEM(data); err != nil {
		return fmt.Errorf("invalid public key file: %w", err)
	}

	if err := os.MkdirAll(keyDir, 0o700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	destPath := PublicKeyPath(name)
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// DeletePrivateKey removes a named private key from the key directory.
func DeletePrivateKey(name string) error {
	if err := validateKeyName(name); err != nil {
		return err
	}
	path := filepath.Join(keyDir, name+privKeyExt)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("private key %q not found", name)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete private key: %w", err)
	}
	return nil
}

// DeletePublicKey removes a named public key from the key directory.
func DeletePublicKey(name string) error {
	if err := validateKeyName(name); err != nil {
		return err
	}
	path := filepath.Join(keyDir, name+pubKeyExt)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("public key %q not found", name)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete public key: %w", err)
	}
	return nil
}

// ListKeys returns information about all keys in the key directory.
func ListKeys() ([]KeyInfo, error) {
	entries, err := os.ReadDir(keyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var keys []KeyInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(keyDir, name)

		if before, ok := strings.CutSuffix(name, privKeyExt); ok {
			keys = append(keys, KeyInfo{
				Name: before,
				Type: "private",
				Path: path,
			})
		} else if before, ok := strings.CutSuffix(name, pubKeyExt); ok {
			keys = append(keys, KeyInfo{
				Name: before,
				Type: "public",
				Path: path,
			})
		}
	}
	return keys, nil
}

// ExportPublicKey reads and returns the PEM-encoded public key with the given name.
// If name is empty, "default" is used.
func ExportPublicKey(name string) ([]byte, error) {
	if name == "" {
		name = "default"
	}
	if err := validateKeyName(name); err != nil {
		return nil, err
	}
	path := PublicKeyPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("public key %q not found", name)
		}
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}
	return data, nil
}

// LoadPrivateKey loads the named private key from the key directory.
func LoadPrivateKey(name string) (crypto.Signer, error) {
	if err := validateKeyName(name); err != nil {
		return nil, err
	}

	path := PrivateKeyPath(name)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat private key: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("private key %q has insecure permissions %v, expected 0600", name, info.Mode().Perm())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("private key does not implement crypto.Signer")
	}
	return signer, nil
}

// LoadAllPublicKeys loads all public keys from the key directory.
func LoadAllPublicKeys() ([]crypto.PublicKey, error) {
	entries, err := os.ReadDir(keyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var keys []crypto.PublicKey
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), pubKeyExt) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(keyDir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read public key %s: %w", e.Name(), err)
		}
		pub, err := parsePublicKeyPEM(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key %s: %w", e.Name(), err)
		}
		keys = append(keys, pub)
	}
	return keys, nil
}

func writePrivateKey(path string, key *ecdsa.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: der,
	}

	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	return nil
}

func writePublicKey(path string, key *ecdsa.PublicKey) error {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}

	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	return nil
}

func parsePublicKeyPEM(data []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	return x509.ParsePKIXPublicKey(block.Bytes)
}
