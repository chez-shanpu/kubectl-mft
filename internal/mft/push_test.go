// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"context"
	"errors"
	"strings"
	"testing"

	"oras.land/oras-go/v2/registry"
)

// Test formatPushError function with different error patterns
func TestFormatPushError(t *testing.T) {
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
				"push permissions",
			},
		},
		{
			name:       "permission error with forbidden keyword",
			inputError: errors.New("push failed: forbidden repository"),
			expectedMsg: []string{
				"access denied to repository docker.io/user/app",
				"push permissions",
			},
		},
		{
			name:       "network error with connection",
			inputError: errors.New("connection refused"),
			expectedMsg: []string{
				"network error while pushing to docker.io",
				"network connection",
			},
		},
		{
			name:       "network error with timeout",
			inputError: errors.New("request timeout"),
			expectedMsg: []string{
				"network error while pushing to docker.io",
				"network connection",
			},
		},
		{
			name:       "network error with network keyword",
			inputError: errors.New("network unreachable"),
			expectedMsg: []string{
				"network error while pushing to docker.io",
				"network connection",
			},
		},
		{
			name:       "general error",
			inputError: errors.New("unknown server error"),
			expectedMsg: []string{
				"failed to push manifest to docker.io/user/app:v1.0.0",
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
			result := formatPushError(tt.inputError, ref)

			if result == nil {
				t.Errorf("formatPushError() returned nil, expected error")
				return
			}

			resultMsg := result.Error()

			// Check that all expected message parts are present
			for _, expected := range tt.expectedMsg {
				if !strings.Contains(resultMsg, expected) {
					t.Errorf("formatPushError() result = %q, expected to contain %q", resultMsg, expected)
				}
			}

			// Ensure the original error is wrapped
			if !strings.Contains(resultMsg, tt.inputError.Error()) {
				t.Errorf("formatPushError() result = %q, expected to contain original error %q", resultMsg, tt.inputError.Error())
			}
		})
	}
}

// Test Push function with invalid inputs (error cases only)
func TestPush_ErrorCases(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		tag         string
		expectedErr string
	}{
		{
			name:        "empty tag",
			tag:         "",
			expectedErr: "failed to parse reference",
		},
		{
			name:        "invalid OCI reference format",
			tag:         "invalid-tag-format",
			expectedErr: "failed to parse reference",
		},
		{
			name:        "tag with special characters",
			tag:         "registry.io/repo:tag@#$%",
			expectedErr: "failed to parse reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Push(ctx, tt.tag)

			if err == nil {
				t.Errorf("Push() expected error but got none for tag %q", tt.tag)
				return
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Push() error = %q, expected to contain %q", err.Error(), tt.expectedErr)
			}
		})
	}
}

// Test createCredentialStore function (error cases)
func TestCreateCredentialStore_ErrorHandling(t *testing.T) {
	// This test checks that the function handles Docker credential store errors gracefully
	// Note: This may pass or fail depending on the Docker setup in the test environment

	t.Run("credential store creation", func(t *testing.T) {
		store, err := createCredentialStore()

		// We can't predict if Docker credentials are available in test environment
		// So we just check that the error (if any) is properly formatted
		if err != nil {
			if !strings.Contains(err.Error(), "failed to create credential store") {
				t.Errorf("createCredentialStore() error = %q, expected to contain 'failed to create credential store'", err.Error())
			}
		} else {
			// If successful, store should not be nil
			if store == nil {
				t.Errorf("createCredentialStore() returned nil store with no error")
			}
		}
	})
}

// Test edge cases for formatPushError
func TestFormatPushError_EdgeCases(t *testing.T) {
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
					t.Errorf("formatPushError() panicked with input %v: %v", tt.inputError, r)
				}
			}()

			result := formatPushError(tt.inputError, ref)

			// Result should not be nil (except for nil input case)
			if tt.inputError != nil && result == nil {
				t.Errorf("formatPushError() returned nil for non-nil input")
			}
		})
	}
}

// Benchmark formatPushError for performance
func BenchmarkFormatPushError(b *testing.B) {
	ref := &registry.Reference{
		Registry:   "docker.io",
		Repository: "user/app",
		Reference:  "v1.0.0",
	}

	testError := errors.New("HTTP 401 Unauthorized access denied")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatPushError(testError, ref)
	}
}
