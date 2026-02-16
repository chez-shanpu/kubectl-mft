// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	kubectlMftPath string
	testRegistry   *Registry
	testFixtures   *Fixtures
	testStorageDir string
	testSchemaDir  string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "kubectl-mft E2E Test Suite")
}

var _ = BeforeSuite(func() {
	By("Building kubectl-mft binary")
	var err error
	kubectlMftPath, err = gexec.Build("github.com/chez-shanpu/kubectl-mft")
	Expect(err).NotTo(HaveOccurred())

	By("Creating test storage directory")
	testStorageDir, err = os.MkdirTemp("", "kubectl-mft-test-storage-*")
	Expect(err).NotTo(HaveOccurred())
	os.Setenv("KUBECTL_MFT_STORAGE_DIR", testStorageDir)

	By("Starting test registry")
	testRegistry = NewRegistry()
	testRegistry.Start()

	By("Creating test schema directory")
	testSchemaDir, err = os.MkdirTemp("", "kubectl-mft-test-schema-*")
	Expect(err).NotTo(HaveOccurred())

	By("Setting up test fixtures")
	testFixtures = NewFixtures()

	By("Configuring test environment")
	SetDefaultEventuallyTimeout(30 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
})

var _ = AfterSuite(func() {
	By("Cleaning up kubectl-mft binary")
	gexec.CleanupBuildArtifacts()

	By("Stopping test registry")
	if testRegistry != nil {
		testRegistry.Stop()
	}

	By("Cleaning up test fixtures")
	if testFixtures != nil {
		testFixtures.Cleanup()
	}

	By("Cleaning up test storage directory")
	if testStorageDir != "" {
		os.RemoveAll(testStorageDir)
	}

	By("Cleaning up test schema directory")
	if testSchemaDir != "" {
		os.RemoveAll(testSchemaDir)
	}
})

// Helper function to execute kubectl-mft command
func ExecuteKubectlMft(args ...string) *gexec.Session {
	cmd := exec.Command(kubectlMftPath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KUBECTL_MFT_STORAGE_DIR=%s", testStorageDir),
		fmt.Sprintf("KUBECTL_MFT_SCHEMA_DIR=%s", testSchemaDir),
	)
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	return session
}

// Helper function to create unique tag reference
func CreateUniqueTag(prefix string) string {
	return fmt.Sprintf("localhost:5000/%s:%d", prefix, time.Now().UnixNano())
}
