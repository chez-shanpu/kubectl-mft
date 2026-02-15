// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// CRDManifest represents a CustomResourceDefinition YAML structure.
// This avoids depending on the k8s.io API packages.
type CRDManifest struct {
	APIVersion string  `yaml:"apiVersion"`
	Kind       string  `yaml:"kind"`
	Spec       CRDSpec `yaml:"spec"`
}

// CRDSpec represents the spec of a CRD.
type CRDSpec struct {
	Group    string       `yaml:"group"`
	Names    CRDNames     `yaml:"names"`
	Versions []CRDVersion `yaml:"versions"`
}

// CRDNames holds the resource kind name.
type CRDNames struct {
	Kind string `yaml:"kind"`
}

// CRDVersion represents a version entry in a CRD spec.
type CRDVersion struct {
	Name   string `yaml:"name"`
	Schema struct {
		OpenAPIV3Schema map[string]any `yaml:"openAPIV3Schema"`
	} `yaml:"schema"`
}

// SchemaInfo holds metadata about a registered CRD schema.
type SchemaInfo struct {
	Group   string `json:"group"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

// schemaIndex is the on-disk index of registered schemas.
type schemaIndex struct {
	Schemas []SchemaInfo `json:"schemas"`
}

// resolveSchemaDir returns the schema directory path.
// It checks KUBECTL_MFT_SCHEMA_DIR env var first, then falls back to default.
func resolveSchemaDir() (string, error) {
	if dir := os.Getenv("KUBECTL_MFT_SCHEMA_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "kubectl-mft", "schemas"), nil
}

// SchemaLocationTemplate returns the kubeconform schema location template
// for the CRD schema directory.
func SchemaLocationTemplate() (string, error) {
	dir, err := resolveSchemaDir()
	if err != nil {
		return "", err
	}
	return dir + "/{{ .Group }}/{{ .ResourceKind }}_{{ .ResourceAPIVersion }}.json", nil
}

// RegisterCRDSchema reads a CRD YAML file and extracts JSON Schema files
// for each version defined in the CRD.
func RegisterCRDSchema(crdFilePath string) error {
	data, err := os.ReadFile(crdFilePath)
	if err != nil {
		return fmt.Errorf("failed to read CRD file: %w", err)
	}

	var crd CRDManifest
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return fmt.Errorf("failed to parse CRD YAML: %w", err)
	}

	if crd.Kind != "CustomResourceDefinition" {
		return fmt.Errorf("expected CustomResourceDefinition, got %q", crd.Kind)
	}

	if crd.Spec.Group == "" || crd.Spec.Names.Kind == "" {
		return fmt.Errorf("CRD is missing required fields (group or kind)")
	}

	if len(crd.Spec.Versions) == 0 {
		return fmt.Errorf("CRD has no versions defined")
	}

	idx, err := loadIndex()
	if err != nil {
		return err
	}

	for _, ver := range crd.Spec.Versions {
		if ver.Schema.OpenAPIV3Schema == nil {
			continue
		}

		if err := saveSchemaFile(crd.Spec.Group, crd.Spec.Names.Kind, ver.Name, ver.Schema.OpenAPIV3Schema); err != nil {
			return fmt.Errorf("failed to save schema for %s/%s %s: %w", crd.Spec.Group, crd.Spec.Names.Kind, ver.Name, err)
		}

		info := SchemaInfo{
			Group:   crd.Spec.Group,
			Kind:    crd.Spec.Names.Kind,
			Version: ver.Name,
		}
		idx.addIfNotExists(info)
	}

	return saveIndex(idx)
}

// ListSchemas returns all registered CRD schemas.
func ListSchemas() ([]SchemaInfo, error) {
	idx, err := loadIndex()
	if err != nil {
		return nil, err
	}
	return idx.Schemas, nil
}

// DeleteSchema removes a registered CRD schema by group and kind.
// It deletes all versions of the specified resource.
func DeleteSchema(group, kind string) error {
	idx, err := loadIndex()
	if err != nil {
		return err
	}

	var remaining []SchemaInfo
	var found bool
	for _, s := range idx.Schemas {
		if s.Group == group && s.Kind == kind {
			found = true
			// Remove the schema file
			filePath, err := schemaFilePath(s.Group, s.Kind, s.Version)
			if err != nil {
				return err
			}
			if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete schema file: %w", err)
			}
		} else {
			remaining = append(remaining, s)
		}
	}

	if !found {
		return fmt.Errorf("schema not found: %s/%s", group, kind)
	}

	// Clean up empty group directory
	dir, err := resolveSchemaDir()
	if err != nil {
		return err
	}
	groupDir := filepath.Join(dir, group)
	entries, err := os.ReadDir(groupDir)
	if err == nil && len(entries) == 0 {
		os.Remove(groupDir)
	}

	idx.Schemas = remaining
	return saveIndex(idx)
}

// ParseGroupKind splits a "group/kind" string into group and kind parts.
func ParseGroupKind(s string) (group, kind string, err error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid format %q: expected <group>/<kind>", s)
	}
	return parts[0], parts[1], nil
}

// saveSchemaFile writes the openAPIV3Schema as a JSON Schema file.
// The file is stored as: <schemaDir>/<group>/<kind_lowercase>_<version>.json
func saveSchemaFile(group, kind, version string, schema map[string]any) error {
	filePath, err := schemaFilePath(group, kind, version)
	if err != nil {
		return err
	}
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create schema directory: %w", err)
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	return os.WriteFile(filePath, data, 0o644)
}

// schemaFilePath returns the file path for a schema.
// kubeconform uses lowercase ResourceKind in templates, so we match that.
func schemaFilePath(group, kind, version string) (string, error) {
	dir, err := resolveSchemaDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, group, fmt.Sprintf("%s_%s.json", strings.ToLower(kind), version)), nil
}

func loadIndex() (*schemaIndex, error) {
	dir, err := resolveSchemaDir()
	if err != nil {
		return nil, err
	}
	indexPath := filepath.Join(dir, "index.json")
	data, err := os.ReadFile(indexPath)
	if os.IsNotExist(err) {
		return &schemaIndex{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read schema index: %w", err)
	}

	var idx schemaIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to parse schema index: %w", err)
	}
	return &idx, nil
}

func saveIndex(idx *schemaIndex) error {
	dir, err := resolveSchemaDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create schema directory: %w", err)
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema index: %w", err)
	}

	return os.WriteFile(filepath.Join(dir, "index.json"), data, 0o644)
}

func (idx *schemaIndex) addIfNotExists(info SchemaInfo) {
	for _, s := range idx.Schemas {
		if s.Group == info.Group && s.Kind == info.Kind && s.Version == info.Version {
			return
		}
	}
	idx.Schemas = append(idx.Schemas, info)
}
