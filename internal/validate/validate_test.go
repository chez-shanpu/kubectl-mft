// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func writeManifestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateManifest_ValidSingleDocument(t *testing.T) {
	dir := t.TempDir()
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  key: value
`
	path := writeManifestFile(t, dir, "valid.yaml", manifest)

	err := ValidateManifest(path)
	if err != nil {
		t.Errorf("expected no error for valid manifest, got: %v", err)
	}
}

func TestValidateManifest_ValidMultiDocument(t *testing.T) {
	dir := t.TempDir()
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
data:
  key: value1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
data:
  key: value2
`
	path := writeManifestFile(t, dir, "multi.yaml", manifest)

	err := ValidateManifest(path)
	if err != nil {
		t.Errorf("expected no error for valid multi-document manifest, got: %v", err)
	}
}

func TestValidateManifest_InvalidManifest(t *testing.T) {
	dir := t.TempDir()
	// spec.replicas should be integer, not string
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
spec:
  replicas: "not-a-number"
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: app
        image: nginx:latest
`
	path := writeManifestFile(t, dir, "invalid.yaml", manifest)

	err := ValidateManifest(path)
	if err == nil {
		t.Error("expected validation error for invalid manifest, got nil")
	}
}

func TestValidateManifest_UnknownField(t *testing.T) {
	dir := t.TempDir()
	// spec.hoge is an unknown field - strict mode (default) should reject this
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: fail
  labels:
    app: fail
spec:
  hoge: yaml
  replicas: 1
  selector:
    matchLabels:
      app: fail
  template:
    metadata:
      labels:
        app: fail
    spec:
      containers:
      - name: nginx
        image: nginx
`
	path := writeManifestFile(t, dir, "unknown-field.yaml", manifest)

	err := ValidateManifest(path)
	if err == nil {
		t.Error("expected validation error for unknown field 'hoge', got nil")
	}
}

func TestValidateManifest_MissingApiVersionKind(t *testing.T) {
	dir := t.TempDir()
	// Document without apiVersion/kind should produce warning, not error
	manifest := `name: some-profile
spec:
  containers:
  - name: debug
    image: busybox
`
	path := writeManifestFile(t, dir, "no-apiversion.yaml", manifest)

	err := ValidateManifest(path)
	if err != nil {
		t.Errorf("expected no error for document without apiVersion/kind (warning only), got: %v", err)
	}
}

func TestValidateManifest_EmptyDocument(t *testing.T) {
	dir := t.TempDir()
	manifest := `---
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
---
`
	path := writeManifestFile(t, dir, "empty-docs.yaml", manifest)

	err := ValidateManifest(path)
	if err != nil {
		t.Errorf("expected no error for manifest with empty documents, got: %v", err)
	}
}

func TestValidateManifest_UnknownCRD(t *testing.T) {
	dir := t.TempDir()
	// Unknown CRD should be skipped (not error) due to IgnoreMissingSchemas
	manifest := `apiVersion: example.com/v1
kind: MyCustomResource
metadata:
  name: test-cr
spec:
  foo: bar
`
	path := writeManifestFile(t, dir, "unknown-crd.yaml", manifest)

	err := ValidateManifest(path)
	if err != nil {
		t.Errorf("expected no error for unknown CRD (should be skipped), got: %v", err)
	}
}

func TestValidateManifest_FileNotFound(t *testing.T) {
	err := ValidateManifest("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestValidateManifest_WithCRDSchema(t *testing.T) {
	schemaDir := t.TempDir()
	t.Setenv("KUBECTL_MFT_SCHEMA_DIR", schemaDir)

	// Create a simple JSON schema for the CRD
	groupDir := filepath.Join(schemaDir, "example.com")
	if err := os.MkdirAll(groupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	schema := `{
  "type": "object",
  "properties": {
    "apiVersion": {"type": "string"},
    "kind": {"type": "string"},
    "metadata": {"type": "object"},
    "spec": {
      "type": "object",
      "properties": {
        "foo": {"type": "string"}
      },
      "required": ["foo"]
    }
  },
  "required": ["apiVersion", "kind", "metadata", "spec"]
}`
	schemaPath := filepath.Join(groupDir, "myresource_v1.json")
	if err := os.WriteFile(schemaPath, []byte(schema), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	manifest := `apiVersion: example.com/v1
kind: MyResource
metadata:
  name: test
spec:
  foo: bar
`
	path := writeManifestFile(t, dir, "crd-resource.yaml", manifest)

	tmpl, err := SchemaLocationTemplate()
	if err != nil {
		t.Fatalf("SchemaLocationTemplate failed: %v", err)
	}
	err = ValidateManifest(path,
		WithSchemaLocations(tmpl),
	)
	if err != nil {
		t.Errorf("expected no error for valid CRD resource with schema, got: %v", err)
	}
}

func TestBuildSchemaLocations(t *testing.T) {
	tests := []struct {
		name      string
		custom    []string
		wantLen   int
		wantFirst string
	}{
		{
			name:      "no custom locations",
			custom:    nil,
			wantLen:   1,
			wantFirst: "default",
		},
		{
			name:      "with custom locations",
			custom:    []string{"/custom/path"},
			wantLen:   2,
			wantFirst: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSchemaLocations(tt.custom)
			if len(result) != tt.wantLen {
				t.Errorf("got %d locations, want %d", len(result), tt.wantLen)
			}
			if result[0] != tt.wantFirst {
				t.Errorf("first location = %q, want %q", result[0], tt.wantFirst)
			}
		})
	}
}
