// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package repository

import (
	"errors"
	"strings"
	"testing"

	"oras.land/oras-go/v2/registry"
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
			name:    "invalid format",
			tag:     "invalid-tag-format",
			wantErr: true,
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

// Test formatRegistryError function with different error patterns
func TestFormatRegistryError(t *testing.T) {
	// Create a sample registry reference for testing
	ref := &registry.Reference{
		Registry:   "docker.io",
		Repository: "user/app",
		Reference:  "v1.0.0",
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
				"failed to access manifest at docker.io/user/app:v1.0.0",
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
			result := formatRegistryError(tt.inputError, ref)

			if result == nil {
				t.Errorf("formatRegistryError() returned nil, expected error")
				return
			}

			resultMsg := result.Error()

			// Check that all expected message parts are present
			for _, expected := range tt.expectedMsg {
				if !strings.Contains(resultMsg, expected) {
					t.Errorf("formatRegistryError() result = %q, expected to contain %q", resultMsg, expected)
				}
			}

			// Ensure the original error is wrapped
			if !strings.Contains(resultMsg, tt.inputError.Error()) {
				t.Errorf("formatRegistryError() result = %q, expected to contain original error %q", resultMsg, tt.inputError.Error())
			}
		})
	}
}

// Test edge cases for formatRegistryError
func TestFormatRegistryError_EdgeCases(t *testing.T) {
	ref := &registry.Reference{
		Registry:   "localhost:5000",
		Repository: "test/app",
		Reference:  "latest",
	}

	tests := []struct {
		name        string
		inputError  error
		description string
	}{
		{
			name:        "nil error - should not happen but test robustness",
			inputError:  nil,
			description: "edge case test",
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
			// This should not panic even with edge case inputs
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("formatRegistryError() panicked with input %v: %v", tt.inputError, r)
				}
			}()

			result := formatRegistryError(tt.inputError, ref)

			// Result should not be nil (except for nil input case)
			if tt.inputError != nil && result == nil {
				t.Errorf("formatRegistryError() returned nil for non-nil input")
			}
		})
	}
}

func TestRepoName(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		repository string
		expected   string
	}{
		{
			name:       "both registry and repository present",
			registry:   "docker.io",
			repository: "user/app",
			expected:   "docker.io/user/app",
		},
		{
			name:       "only registry present",
			registry:   "gcr.io",
			repository: "",
			expected:   "gcr.io",
		},
		{
			name:       "only repository present",
			registry:   "",
			repository: "myapp",
			expected:   "myapp",
		},
		{
			name:       "both empty",
			registry:   "",
			repository: "",
			expected:   "",
		},
		{
			name:       "nested repository path",
			registry:   "gcr.io",
			repository: "project/team/service",
			expected:   "gcr.io/project/team/service",
		},
		{
			name:       "registry with subdomain",
			registry:   "us-west2-docker.pkg.dev",
			repository: "project/repo",
			expected:   "us-west2-docker.pkg.dev/project/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &registry.Reference{
				Registry:   tt.registry,
				Repository: tt.repository,
			}

			result := repoName(ref)
			if result != tt.expected {
				t.Errorf("repoName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
