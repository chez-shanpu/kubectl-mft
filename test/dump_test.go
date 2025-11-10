// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Dump Command", func() {

	var manifestPath string
	var testTag string

	BeforeEach(func() {
		manifestPath = testFixtures.CreateManifestFile("test-deployment.yaml", testFixtures.GetSimpleManifest())
		testTag = CreateUniqueTag("dump-test")
	})

	AfterEach(func() {
		session := ExecuteKubectlMft("delete", "-t", testTag, "--force")
		Eventually(session, 10*time.Second).Should(gexec.Exit(0))
	})

	Context("when dumping manifest to stdout", func() {
		It("should successfully dump the packed manifest", func() {
			By("First packing the manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, "-t", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Then dumping the manifest to stdout")
			session = ExecuteKubectlMft("dump", "-t", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying the output matches the original manifest")
			output := string(session.Out.Contents())
			Expect(output).To(Equal(testFixtures.GetSimpleManifest()))
		})
	})

	Context("when dumping manifest to file", func() {
		It("should successfully dump the manifest to specified file", func() {
			By("First packing the manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, "-t", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Then dumping to output file")
			outputPath := filepath.Join(testFixtures.GetTempDir(), "dumped.yaml")
			session = ExecuteKubectlMft("dump", "-t", testTag, "-o", outputPath)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying the output file exists and matches the original manifest")
			Expect(outputPath).To(BeAnExistingFile())
			content, err := os.ReadFile(outputPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal(testFixtures.GetSimpleManifest()))
		})

		It("should overwrite existing file", func() {
			By("First packing the manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, "-t", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Creating an existing file")
			outputPath := filepath.Join(testFixtures.GetTempDir(), "existing.yaml")
			err := os.WriteFile(outputPath, []byte("old content"), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("Dumping to the existing file")
			session = ExecuteKubectlMft("dump", "-t", testTag, "-o", outputPath)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying the file was overwritten")
			content, err := os.ReadFile(outputPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).NotTo(ContainSubstring("old content"))
			Expect(string(content)).To(ContainSubstring("kind: Deployment"))
		})
	})

	Context("when dumping non-existent manifest", func() {
		It("should fail with appropriate error message", func() {
			nonExistentTag := CreateUniqueTag("non-existent")
			session := ExecuteKubectlMft("dump", "-t", nonExistentTag)
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("failed to resolve reference"))
		})
	})

	Context("when tag format is invalid", func() {
		It("should fail with appropriate error message", func() {
			session := ExecuteKubectlMft("dump", "-t", "invalid-tag-format")
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("invalid reference"))
		})
	})

	Context("when tag flag is missing", func() {
		It("should fail with appropriate error message", func() {
			session := ExecuteKubectlMft("dump")
			Eventually(session).Should(gexec.Exit(1))
		})
	})
})
