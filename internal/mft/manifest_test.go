// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package mft

import (
	"testing"

	"oras.land/oras-go/v2/registry"
)

func TestManifestDIRName(t *testing.T) {
	tests := []struct {
		name       string
		registry   string
		repository string
		reference  string
		expected   string
	}{
		{
			name:       "localhost reference",
			registry:   "localhost",
			repository: "test",
			reference:  "v1.0.0",
			expected:   "localhost-test-v1.0.0",
		},
		{
			name:       "docker hub with slash",
			registry:   "docker.io",
			repository: "user/app",
			reference:  "latest",
			expected:   "docker.io-user-app-latest",
		},
		{
			name:       "nested repository",
			registry:   "gcr.io",
			repository: "project/team/service",
			reference:  "v2.1.0",
			expected:   "gcr.io-project-team-service-v2.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := &registry.Reference{
				Registry:   tt.registry,
				Repository: tt.repository,
				Reference:  tt.reference,
			}

			result := manifestName(ref)
			if result != tt.expected {
				t.Errorf("manifestName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
