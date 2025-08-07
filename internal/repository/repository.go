// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package repository

import (
	"context"
	"fmt"
	"strings"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry"
)

// ParseReference parses and validates the OCI reference
func ParseReference(tag string) (*registry.Reference, error) {
	ref, err := registry.ParseReference(tag)
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference %q: %w", tag, err)
	}
	return &ref, nil
}

// Copy handles copying manifests between OCI targets
func Copy(ctx context.Context, source oras.Target, dest oras.Target, ref *registry.Reference) error {
	_, err := oras.Copy(ctx, source, ref.ReferenceOrDefault(), dest, ref.ReferenceOrDefault(), oras.DefaultCopyOptions)
	if err != nil {
		return formatRegistryError(err, ref)
	}
	return nil
}

// formatRegistryError provides better error messages based on common registry operation failures
func formatRegistryError(err error, ref *registry.Reference) error {
	if err == nil {
		return fmt.Errorf("unknown error occurred with %s/%s:%s",
			ref.Registry, ref.Repository, ref.ReferenceOrDefault())
	}

	errorMsg := err.Error()

	// Check for common error patterns and provide helpful messages
	if strings.Contains(errorMsg, "401") || strings.Contains(errorMsg, "unauthorized") {
		return fmt.Errorf("authentication failed for registry %s: %w\n"+
			"Please ensure you are logged in using 'docker login %s'", ref.Registry, err, ref.Registry)
	}

	if strings.Contains(errorMsg, "403") || strings.Contains(errorMsg, "forbidden") {
		return fmt.Errorf("access denied to repository %s/%s: %w\n"+
			"Check if you have the required permissions to this repository", ref.Registry, ref.Repository, err)
	}

	if strings.Contains(errorMsg, "connection") || strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "network") {
		return fmt.Errorf("network error with %s: %w\n"+
			"Check your network connection and registry availability", ref.Registry, err)
	}

	return fmt.Errorf("failed to access manifest at %s/%s:%s: %w",
		ref.Registry, ref.Repository, ref.ReferenceOrDefault(), err)
}
