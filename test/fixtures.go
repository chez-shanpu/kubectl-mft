// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/gomega"
)

// Fixtures manages test data and temporary directories
type Fixtures struct {
	tempDir string
}

// NewFixtures creates a new test fixtures instance
func NewFixtures() *Fixtures {
	tempDir, err := os.MkdirTemp("", "kubectl-mft-test-*")
	Expect(err).NotTo(HaveOccurred())

	return &Fixtures{
		tempDir: tempDir,
	}
}

// Cleanup removes all temporary test files
func (f *Fixtures) Cleanup() {
	if f.tempDir != "" {
		os.RemoveAll(f.tempDir)
	}
}

// GetTempDir returns a temporary directory for test operations
func (f *Fixtures) GetTempDir() string {
	return f.tempDir
}

// CreateManifestFile creates a manifest file with the given content
func (f *Fixtures) CreateManifestFile(name, content string) string {
	filePath := filepath.Join(f.tempDir, name)
	err := os.WriteFile(filePath, []byte(content), 0644)
	Expect(err).NotTo(HaveOccurred())
	return filePath
}

// GetSimpleManifest returns a simple Deployment manifest content
func (f *Fixtures) GetSimpleManifest() string {
	return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-app
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-app
  template:
    metadata:
      labels:
        app: test-app
    spec:
      containers:
      - name: app
        image: nginx:latest
        ports:
        - containerPort: 80`
}

// GetComplexManifest returns multiple manifests content
func (f *Fixtures) GetComplexManifest() string {
	return `apiVersion: v1
kind: ConfigMap
metadata:
  name: complex-config
data:
  key: value
---
apiVersion: v1
kind: Service
metadata:
  name: test-service
spec:
  selector:
    app: test
  ports:
  - port: 80
    targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
spec:
  replicas: 2
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
        ports:
        - containerPort: 8080`
}
