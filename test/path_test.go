// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Path Command", func() {
	var manifestPath string

	BeforeEach(func() {
		manifestPath = testFixtures.CreateManifestFile("test-deployment.yaml", testFixtures.GetSimpleManifest())
	})

	Context("when getting path for packed manifest", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("path-test")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should successfully return the blob path", func() {
			session := ExecuteKubectlMft("path", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			output := strings.TrimSpace(string(session.Out.Contents()))
			Expect(output).NotTo(BeEmpty())
			Expect(output).To(ContainSubstring(testStorageDir))
		})

		It("should return a path that exists", func() {
			session := ExecuteKubectlMft("path", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			blobPath := strings.TrimSpace(string(session.Out.Contents()))
			Expect(blobPath).To(BeAnExistingFile())
		})

		It("should return path in OCI layout structure", func() {
			session := ExecuteKubectlMft("path", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			blobPath := strings.TrimSpace(string(session.Out.Contents()))

			// Path should contain "blobs" directory
			Expect(blobPath).To(ContainSubstring("blobs"))

			// Path should contain algorithm directory (e.g., sha256)
			Expect(blobPath).To(MatchRegexp(`blobs/[a-z0-9]+/[a-f0-9]{64}`))
		})

		It("should point to correct blob file with manifest content", func() {
			session := ExecuteKubectlMft("path", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			blobPath := strings.TrimSpace(string(session.Out.Contents()))

			By("Reading the blob file")
			content, err := os.ReadFile(blobPath)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying it contains the manifest content")
			Expect(string(content)).To(ContainSubstring("kind: Deployment"))
			Expect(string(content)).To(ContainSubstring("name: test-app"))
		})

		It("should be consistent across multiple calls", func() {
			session1 := ExecuteKubectlMft("path", testTag)
			Eventually(session1, 10*time.Second).Should(gexec.Exit(0))
			path1 := strings.TrimSpace(string(session1.Out.Contents()))

			session2 := ExecuteKubectlMft("path", testTag)
			Eventually(session2, 10*time.Second).Should(gexec.Exit(0))
			path2 := strings.TrimSpace(string(session2.Out.Contents()))

			Expect(path1).To(Equal(path2))
		})
	})

	Context("when getting path for non-existent tag", func() {
		It("should fail with appropriate error message", func() {
			nonExistentTag := CreateUniqueTag("non-existent")
			session := ExecuteKubectlMft("path", nonExistentTag)
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("failed to resolve reference"))
		})
	})

	Context("when simple tag does not exist", func() {
		It("should fail with not found error", func() {
			session := ExecuteKubectlMft("path", "nonexistent-simple-tag")
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("not found"))
		})
	})

	Context("when tag argument is missing", func() {
		It("should fail with appropriate error message", func() {
			session := ExecuteKubectlMft("path")
			Eventually(session).Should(gexec.Exit(1))
		})
	})

	Context("when verifying path structure for different repositories", func() {
		var repo1Tag, repo2Tag string

		BeforeEach(func() {
			repo1Tag = "localhost:5000/repo-a:v1.0.0"
			repo2Tag = "localhost:5000/repo-b:v1.0.0"

			session := ExecuteKubectlMft("pack", "-f", manifestPath, repo1Tag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("pack", "-f", manifestPath, repo2Tag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", repo1Tag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("delete", repo2Tag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should return paths in different repository directories", func() {
			session1 := ExecuteKubectlMft("path", repo1Tag)
			Eventually(session1, 10*time.Second).Should(gexec.Exit(0))
			path1 := strings.TrimSpace(string(session1.Out.Contents()))

			session2 := ExecuteKubectlMft("path", repo2Tag)
			Eventually(session2, 10*time.Second).Should(gexec.Exit(0))
			path2 := strings.TrimSpace(string(session2.Out.Contents()))

			// Paths should be in different repository directories
			Expect(path1).To(ContainSubstring("repo-a"))
			Expect(path2).To(ContainSubstring("repo-b"))
		})
	})

	Context("when verifying path for multiple tags in same repository", func() {
		var baseRepo string
		var tag1, tag2 string
		var manifest1, manifest2 string

		BeforeEach(func() {
			baseRepo = "localhost:5000/same-repo-path"
			tag1 = baseRepo + ":v1.0.0"
			tag2 = baseRepo + ":v2.0.0"

			// Create two different manifests
			manifest1 = testFixtures.CreateManifestFile("manifest1.yaml", testFixtures.GetSimpleManifest())
			manifest2 = testFixtures.CreateManifestFile("manifest2.yaml", testFixtures.GetComplexManifest())

			session := ExecuteKubectlMft("pack", "-f", manifest1, tag1)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("pack", "-f", manifest2, tag2)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", tag1, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("delete", tag2, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should return different blob paths for different content", func() {
			session1 := ExecuteKubectlMft("path", tag1)
			Eventually(session1, 10*time.Second).Should(gexec.Exit(0))
			path1 := strings.TrimSpace(string(session1.Out.Contents()))

			session2 := ExecuteKubectlMft("path", tag2)
			Eventually(session2, 10*time.Second).Should(gexec.Exit(0))
			path2 := strings.TrimSpace(string(session2.Out.Contents()))

			// Different content should have different blob paths
			Expect(path1).NotTo(Equal(path2))

			// Both should exist
			Expect(path1).To(BeAnExistingFile())
			Expect(path2).To(BeAnExistingFile())

			// Both should be in the same repository directory
			Expect(filepath.Dir(filepath.Dir(path1))).To(Equal(filepath.Dir(filepath.Dir(path2))))
		})

		It("should have same blob path for identical content", func() {
			// Pack the same manifest with different tag
			tag3 := baseRepo + ":v3.0.0"
			session := ExecuteKubectlMft("pack", "-f", manifest1, tag3)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session1 := ExecuteKubectlMft("path", tag1)
			Eventually(session1, 10*time.Second).Should(gexec.Exit(0))
			path1 := strings.TrimSpace(string(session1.Out.Contents()))

			session3 := ExecuteKubectlMft("path", tag3)
			Eventually(session3, 10*time.Second).Should(gexec.Exit(0))
			path3 := strings.TrimSpace(string(session3.Out.Contents()))

			// Same content should point to same blob
			Expect(path1).To(Equal(path3))

			session = ExecuteKubectlMft("delete", tag3, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})
})
