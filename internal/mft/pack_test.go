package mft

import (
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

			result := manifestDIRName(ref)
			if result != tt.expected {
				t.Errorf("manifestDIRName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
