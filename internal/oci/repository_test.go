package oci

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oras.land/oras-go/v2/errdef"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		wantErr bool
	}{
		{
			name:    "valid localhost reference",
			tag:     "localhost/test:v1.0.0",
			wantErr: false,
		},
		{
			name:    "valid docker hub reference",
			tag:     "docker.io/user/repo:latest",
			wantErr: false,
		},
		{
			name:    "empty tag",
			tag:     "",
			wantErr: true,
		},
		{
			name:    "simple tag name without slash (normalized to local/)",
			tag:     "myapp:v1.0.0",
			wantErr: false,
		},
		{
			name:    "simple tag name without version",
			tag:     "myapp",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := parseReference(tt.tag)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseReference() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("parseReference() unexpected error: %v", err)
				}
				if ref.String() == "" {
					t.Errorf("parseReference() returned empty reference")
				}
			}
		})
	}
}

// Test formatCopyError method with different error patterns
func TestFormatCopyError(t *testing.T) {
	t.Run("ErrorPatterns", func(t *testing.T) {
		// Create a sample repository for testing
		repo, err := NewRepository("docker.io/user/app:v1.0.0")
		if err != nil {
			t.Fatalf("Failed to create test repository: %v", err)
		}

		tests := []struct {
			name        string
			inputError  error
			expectedMsg []string // Multiple strings that should be in the result
		}{
			{
				name:       "authentication error with 401",
				inputError: errors.New("HTTP 401 Unauthorized"),
				expectedMsg: []string{
					"authentication failed for registry docker.io",
					"docker login docker.io",
				},
			},
			{
				name:       "authentication error with unauthorized keyword",
				inputError: errors.New("push failed: unauthorized access"),
				expectedMsg: []string{
					"authentication failed for registry docker.io",
					"docker login docker.io",
				},
			},
			{
				name:       "permission error with 403",
				inputError: errors.New("HTTP 403 Forbidden"),
				expectedMsg: []string{
					"access denied to repository docker.io/user/app",
					"required permissions",
				},
			},
			{
				name:       "permission error with forbidden keyword",
				inputError: errors.New("push failed: forbidden repository"),
				expectedMsg: []string{
					"access denied to repository docker.io/user/app",
					"required permissions",
				},
			},
			{
				name:       "network error with connection",
				inputError: errors.New("connection refused"),
				expectedMsg: []string{
					"network error with docker.io",
					"network connection",
				},
			},
			{
				name:       "network error with timeout",
				inputError: errors.New("request timeout"),
				expectedMsg: []string{
					"network error with docker.io",
					"network connection",
				},
			},
			{
				name:       "network error with network keyword",
				inputError: errors.New("network unreachable"),
				expectedMsg: []string{
					"network error with docker.io",
					"network connection",
				},
			},
			{
				name:       "general error",
				inputError: errors.New("unknown server error"),
				expectedMsg: []string{
					"failed to copy manifest docker.io/user/app:v1.0.0",
					"unknown server error",
				},
			},
			{
				name:       "multiple keyword match - auth takes precedence",
				inputError: errors.New("401 unauthorized connection failed"),
				expectedMsg: []string{
					"authentication failed for registry docker.io",
					"docker login",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := repo.formatCopyError(tt.inputError)

				if result == nil {
					t.Errorf("formatCopyError() returned nil, expected error")
					return
				}

				resultMsg := result.Error()

				// Check that all expected message parts are present
				for _, expected := range tt.expectedMsg {
					if !strings.Contains(resultMsg, expected) {
						t.Errorf("formatCopyError() result = %q, expected to contain %q", resultMsg, expected)
					}
				}

				// Ensure the original error is wrapped
				if !strings.Contains(resultMsg, tt.inputError.Error()) {
					t.Errorf("formatCopyError() result = %q, expected to contain original error %q", resultMsg, tt.inputError.Error())
				}
			})
		}
	})

	t.Run("BoundaryConditions", func(t *testing.T) {
		repo, err := NewRepository("localhost:5000/test/app:latest")
		if err != nil {
			t.Fatalf("Failed to create test repository: %v", err)
		}

		tests := []struct {
			name        string
			inputError  error
			description string
		}{
			{
				name:        "nil error",
				inputError:  nil,
				description: "nil error should be handled",
			},
			{
				name:        "empty error message",
				inputError:  errors.New(""),
				description: "empty error string",
			},
			{
				name:        "very long error message",
				inputError:  errors.New(strings.Repeat("error ", 100)),
				description: "long error message handling",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// This should not panic with boundary condition inputs
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("formatCopyError() panicked with input %v: %v", tt.inputError, r)
					}
				}()

				result := repo.formatCopyError(tt.inputError)

				// Result should not be nil
				if result == nil {
					t.Errorf("formatCopyError() returned nil")
				}
			})
		}
	})
}

func TestRepositoryName(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "both registry and repository present",
			tag:      "docker.io/user/app:latest",
			expected: "docker.io/user/app",
		},
		{
			name:     "nested repository path",
			tag:      "gcr.io/project/team/service:v1.0.0",
			expected: "gcr.io/project/team/service",
		},
		{
			name:     "registry with subdomain",
			tag:      "us-west2-docker.pkg.dev/project/repo:latest",
			expected: "us-west2-docker.pkg.dev/project/repo",
		},
		{
			name:     "localhost with port",
			tag:      "localhost:5000/myapp:dev",
			expected: "localhost:5000/myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewRepository(tt.tag)
			if err != nil {
				t.Fatalf("NewRepository() failed: %v", err)
			}

			result := repo.Name()
			if result != tt.expected {
				t.Errorf("Repository.Name() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "simple tag without slash",
			tag:      "myapp:v1.0.0",
			expected: "local/myapp:v1.0.0",
		},
		{
			name:     "simple tag without version",
			tag:      "myapp",
			expected: "local/myapp",
		},
		{
			name:     "tag with registry (not normalized)",
			tag:      "docker.io/user/app:latest",
			expected: "docker.io/user/app:latest",
		},
		{
			name:     "localhost tag (not normalized)",
			tag:      "localhost/test:v1",
			expected: "localhost/test:v1",
		},
		{
			name:     "tag with single slash",
			tag:      "repo/app:v1",
			expected: "repo/app:v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTag(tt.tag)
			if result != tt.expected {
				t.Errorf("normalizeTag(%q) = %q, expected %q", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestCopyDifferentTag(t *testing.T) {
	// Override baseDir to use a temp directory
	origBaseDir := baseDir
	baseDir = t.TempDir()
	t.Cleanup(func() { baseDir = origBaseDir })

	ctx := context.Background()

	// Create a test manifest file
	manifestFile := filepath.Join(t.TempDir(), "test.yaml")
	if err := os.WriteFile(manifestFile, []byte("apiVersion: v1\nkind: ConfigMap\n"), 0o644); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	// Save a manifest with source tag
	srcRepo, err := NewRepository("myrepo:src")
	if err != nil {
		t.Fatalf("NewRepository(src) failed: %v", err)
	}
	if err := srcRepo.Save(ctx, manifestFile); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Copy to a different tag in a different repository
	if err := srcRepo.Copy(ctx, "otherrepo:dest"); err != nil {
		t.Fatalf("Copy() failed: %v", err)
	}

	// Verify dest tag is resolvable in the dest repository
	destRepo, err := NewRepository("otherrepo:dest")
	if err != nil {
		t.Fatalf("NewRepository(dest) failed: %v", err)
	}
	destStore, err := destRepo.newOCILayoutStore()
	if err != nil {
		t.Fatalf("newOCILayoutStore(dest) failed: %v", err)
	}
	if _, err := destStore.Resolve(ctx, "dest"); err != nil {
		t.Errorf("dest tag should be resolvable, got error: %v", err)
	}

	// Verify source tag does NOT exist in the dest repository
	if _, err := destStore.Resolve(ctx, "src"); !errors.Is(err, errdef.ErrNotFound) {
		t.Errorf("source tag should not exist in dest repo, got: %v", err)
	}
}

func TestCopyDuplicateTagError(t *testing.T) {
	origBaseDir := baseDir
	baseDir = t.TempDir()
	t.Cleanup(func() { baseDir = origBaseDir })

	ctx := context.Background()

	manifestFile := filepath.Join(t.TempDir(), "test.yaml")
	if err := os.WriteFile(manifestFile, []byte("test: data\n"), 0o644); err != nil {
		t.Fatalf("failed to create test manifest: %v", err)
	}

	srcRepo, err := NewRepository("myrepo:v1")
	if err != nil {
		t.Fatalf("NewRepository(src) failed: %v", err)
	}
	if err := srcRepo.Save(ctx, manifestFile); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Save the same manifest to dest tag so the copy will fail with "already exists"
	destRepo, err := NewRepository("myrepo:v2")
	if err != nil {
		t.Fatalf("NewRepository(dest) failed: %v", err)
	}
	if err := destRepo.Save(ctx, manifestFile); err != nil {
		t.Fatalf("Save(dest) failed: %v", err)
	}

	// Copy should fail because dest tag already exists
	err = srcRepo.Copy(ctx, "myrepo:v2")
	if err == nil {
		t.Fatal("Copy() should have failed when dest tag already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestIsLocalRegistry(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		expected bool
	}{
		{
			name:     "localhost without port",
			registry: "localhost",
			expected: true,
		},
		{
			name:     "localhost with port",
			registry: "localhost:5000",
			expected: true,
		},
		{
			name:     "localhost with path",
			registry: "localhost/repo",
			expected: true,
		},
		{
			name:     "127.0.0.1 without port",
			registry: "127.0.0.1",
			expected: true,
		},
		{
			name:     "127.0.0.1 with port",
			registry: "127.0.0.1:8080",
			expected: true,
		},
		{
			name:     "remote registry",
			registry: "docker.io",
			expected: false,
		},
		{
			name:     "gcr.io",
			registry: "gcr.io",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalRegistry(tt.registry)
			if result != tt.expected {
				t.Errorf("isLocalRegistry(%q) = %v, expected %v", tt.registry, result, tt.expected)
			}
		})
	}
}
