// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestSchemaDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("KUBECTL_MFT_SCHEMA_DIR", dir)
	return dir
}

func writeCRDFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const testCRDYAML = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: myresources.example.com
spec:
  group: example.com
  names:
    kind: MyResource
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
          spec:
            type: object
            properties:
              replicas:
                type: integer
              name:
                type: string
            required:
            - name
  - name: v2
    schema:
      openAPIV3Schema:
        type: object
        properties:
          apiVersion:
            type: string
          kind:
            type: string
          metadata:
            type: object
          spec:
            type: object
            properties:
              replicas:
                type: integer
              name:
                type: string
              description:
                type: string
            required:
            - name
`

func TestRegisterCRDSchema(t *testing.T) {
	schemaDir := setupTestSchemaDir(t)

	crdDir := t.TempDir()
	crdPath := writeCRDFile(t, crdDir, "crd.yaml", testCRDYAML)

	err := RegisterCRDSchema(crdPath)
	if err != nil {
		t.Fatalf("RegisterCRDSchema failed: %v", err)
	}

	// Verify schema files were created
	v1Schema := filepath.Join(schemaDir, "example.com", "myresource_v1.json")
	if _, err := os.Stat(v1Schema); os.IsNotExist(err) {
		t.Error("v1 schema file was not created")
	}

	v2Schema := filepath.Join(schemaDir, "example.com", "myresource_v2.json")
	if _, err := os.Stat(v2Schema); os.IsNotExist(err) {
		t.Error("v2 schema file was not created")
	}

	// Verify index was updated
	schemas, err := ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas failed: %v", err)
	}
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}
}

func TestRegisterCRDSchema_NotCRD(t *testing.T) {
	setupTestSchemaDir(t)

	crdDir := t.TempDir()
	content := `apiVersion: v1
kind: ConfigMap
metadata:
  name: not-a-crd
`
	crdPath := writeCRDFile(t, crdDir, "configmap.yaml", content)

	err := RegisterCRDSchema(crdPath)
	if err == nil {
		t.Error("expected error for non-CRD resource")
	}
}

func TestRegisterCRDSchema_Idempotent(t *testing.T) {
	setupTestSchemaDir(t)

	crdDir := t.TempDir()
	crdPath := writeCRDFile(t, crdDir, "crd.yaml", testCRDYAML)

	// Register twice
	if err := RegisterCRDSchema(crdPath); err != nil {
		t.Fatalf("first RegisterCRDSchema failed: %v", err)
	}
	if err := RegisterCRDSchema(crdPath); err != nil {
		t.Fatalf("second RegisterCRDSchema failed: %v", err)
	}

	schemas, err := ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas failed: %v", err)
	}
	// Should not have duplicate entries
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas (v1, v2), got %d", len(schemas))
	}
}

func TestListSchemas_Empty(t *testing.T) {
	setupTestSchemaDir(t)

	schemas, err := ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas failed: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(schemas))
	}
}

func TestDeleteSchema(t *testing.T) {
	schemaDir := setupTestSchemaDir(t)

	crdDir := t.TempDir()
	crdPath := writeCRDFile(t, crdDir, "crd.yaml", testCRDYAML)

	if err := RegisterCRDSchema(crdPath); err != nil {
		t.Fatalf("RegisterCRDSchema failed: %v", err)
	}

	err := DeleteSchema("example.com", "MyResource")
	if err != nil {
		t.Fatalf("DeleteSchema failed: %v", err)
	}

	// Verify schemas were removed
	schemas, err := ListSchemas()
	if err != nil {
		t.Fatalf("ListSchemas failed: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("expected 0 schemas after delete, got %d", len(schemas))
	}

	// Verify schema files were deleted
	v1Schema := filepath.Join(schemaDir, "example.com", "myresource_v1.json")
	if _, err := os.Stat(v1Schema); !os.IsNotExist(err) {
		t.Error("v1 schema file was not deleted")
	}
}

func TestDeleteSchema_NotFound(t *testing.T) {
	setupTestSchemaDir(t)

	err := DeleteSchema("nonexistent.io", "Foo")
	if err == nil {
		t.Error("expected error for non-existent schema")
	}
}

func TestParseGroupKind(t *testing.T) {
	tests := []struct {
		input     string
		wantGroup string
		wantKind  string
		wantErr   bool
	}{
		{"cilium.io/CiliumNetworkPolicy", "cilium.io", "CiliumNetworkPolicy", false},
		{"example.com/MyResource", "example.com", "MyResource", false},
		{"noslash", "", "", true},
		{"/noempty", "", "", true},
		{"noempty/", "", "", true},
		{"", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			group, kind, err := ParseGroupKind(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGroupKind(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
				return
			}
			if group != tt.wantGroup {
				t.Errorf("group = %q, want %q", group, tt.wantGroup)
			}
			if kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", kind, tt.wantKind)
			}
		})
	}
}
