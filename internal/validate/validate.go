// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package validate

import (
	"fmt"
	"os"
	"strings"

	"github.com/yannh/kubeconform/pkg/validator"
)

// options holds the configuration for manifest validation.
type options struct {
	schemaLocations []string
}

// Option configures the manifest validation behavior.
type Option func(*options)

// WithSchemaLocations sets custom schema locations for CRD validation.
func WithSchemaLocations(locations ...string) Option {
	return func(o *options) {
		o.schemaLocations = append(o.schemaLocations, locations...)
	}
}

// ValidateManifest validates a Kubernetes manifest file using kubeconform.
// It supports multi-document YAML (separated by ---) and validates each document individually.
// Documents without apiVersion/kind (e.g. debug container profiles) produce warnings, not errors.
// Resources with missing schemas (unregistered CRDs) are skipped.
func ValidateManifest(manifestPath string, opts ...Option) error {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	schemaLocations := buildSchemaLocations(o.schemaLocations)

	v, err := validator.New(schemaLocations, validator.Opts{
		Strict:               true,
		IgnoreMissingSchemas: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}

	f, err := os.Open(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to open manifest file: %w", err)
	}
	defer f.Close()

	results := v.Validate(manifestPath, f)

	var invalidErrors []string
	for _, res := range results {
		switch res.Status {
		case validator.Valid:
			// Validation passed
		case validator.Invalid:
			msg := formatInvalidResult(res)
			invalidErrors = append(invalidErrors, msg)
		case validator.Error:
			// Parse errors (e.g. missing apiVersion/kind) are treated as warnings
			// to support debug container profiles and other non-standard formats
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", manifestPath, res.Err)
		case validator.Skipped:
			// Resource skipped due to missing schema (unregistered CRD)
			fmt.Fprintf(os.Stderr, "info: %s: resource skipped (no schema found)\n", manifestPath)
		case validator.Empty:
			// Empty document, skip
		}
	}

	if len(invalidErrors) > 0 {
		return fmt.Errorf("\n%s", strings.Join(invalidErrors, "\n"))
	}

	return nil
}

// buildSchemaLocations constructs the full list of schema locations.
// It always includes the default Kubernetes schemas and appends any custom locations.
func buildSchemaLocations(custom []string) []string {
	locations := []string{"default"}
	locations = append(locations, custom...)
	return locations
}

// formatInvalidResult formats a validation error result into a human-readable message.
// When ValidationErrors are present, only those are shown (res.Err contains redundant
// schema URL information). res.Err is used as a fallback when ValidationErrors is empty.
func formatInvalidResult(res validator.Result) string {
	var parts []string
	if len(res.ValidationErrors) > 0 {
		for _, ve := range res.ValidationErrors {
			if ve.Path != "" {
				parts = append(parts, fmt.Sprintf("  - %s: %s", ve.Path, ve.Msg))
			} else {
				parts = append(parts, fmt.Sprintf("  - %s", ve.Msg))
			}
		}
	} else if res.Err != nil {
		parts = append(parts, res.Err.Error())
	}
	return strings.Join(parts, "\n")
}
