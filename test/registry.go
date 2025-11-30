// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// Registry manages a local Docker registry for testing
type Registry struct {
	port      string
	container string
	tempDir   string
	session   *gexec.Session
}

// NewRegistry creates a new Registry instance
func NewRegistry() *Registry {
	port := findFreePort()
	containerName := fmt.Sprintf("kubectl-mft-test-registry-%d", time.Now().UnixNano())
	tempDir := filepath.Join(os.TempDir(), containerName)

	return &Registry{
		port:      port,
		container: containerName,
		tempDir:   tempDir,
	}
}

// Start starts the Docker registry container
func (r *Registry) Start() {
	// Create temporary directory for registry data
	err := os.MkdirAll(r.tempDir, 0o755)
	Expect(err).NotTo(HaveOccurred())

	cmd := exec.Command("docker", "run", "--rm",
		"--name", r.container,
		"-p", fmt.Sprintf("%s:5000", r.port),
		"-v", fmt.Sprintf("%s:/var/lib/registry", r.tempDir),
		"registry:3")

	r.session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	r.waitForReady()
}

// Stop stops the registry container
func (r *Registry) Stop() {
	if r.session != nil {
		r.session.Signal(syscall.SIGTERM)
		Eventually(r.session, 10*time.Second).Should(gexec.Exit())
	}

	// Clean up container if it still exists
	stopCmd := exec.Command("docker", "stop", r.container)
	_ = stopCmd.Run()

	rmCmd := exec.Command("docker", "rm", "-f", r.container)
	_ = rmCmd.Run()
}

// GetRegistryURL returns the registry URL
func (r *Registry) GetRegistryURL() string {
	return fmt.Sprintf("localhost:%s", r.port)
}

// waitForReady waits for the registry to be ready to accept connections
func (r *Registry) waitForReady() {
	Eventually(func() bool {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%s", r.port), 1*time.Second)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, 30*time.Second, 1*time.Second).Should(BeTrue())

	// Additional wait to ensure registry is fully initialized
	time.Sleep(2 * time.Second)
}

// findFreePort finds an available port for the registry
func findFreePort() string {
	listener, err := net.Listen("tcp", ":0")
	Expect(err).NotTo(HaveOccurred())
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("%d", port)
}
