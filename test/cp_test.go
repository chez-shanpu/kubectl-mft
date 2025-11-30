// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Copy Command", func() {

	var manifestPath string
	var sourceTag string

	BeforeEach(func() {
		manifestPath = testFixtures.CreateManifestFile("test-deployment.yaml", testFixtures.GetSimpleManifest())
		sourceTag = CreateUniqueTag("cp-test-source")

		By("Packing source manifest")
		session := ExecuteKubectlMft("pack", "-f", manifestPath, "-t", sourceTag)
		Eventually(session, 30*time.Second).Should(gexec.Exit(0))
	})

	AfterEach(func() {
		// Clean up source tag
		session := ExecuteKubectlMft("delete", "-t", sourceTag, "--force")
		Eventually(session, 10*time.Second).Should(gexec.Exit(0))
	})

	Context("Basic copy operations", func() {
		It("should copy manifest to new tag in same repository", func() {
			destTag := fmt.Sprintf("%s-copy", sourceTag)

			By("Copying manifest to new tag")
			session := ExecuteKubectlMft("cp", sourceTag, destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying the copy succeeded silently (no output)")
			Expect(session.Out.Contents()).To(BeEmpty())

			By("Verifying source manifest still exists")
			session = ExecuteKubectlMft("dump", "-t", sourceTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			sourceContent := string(session.Out.Contents())

			By("Verifying destination manifest exists")
			session = ExecuteKubectlMft("dump", "-t", destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			destContent := string(session.Out.Contents())

			By("Verifying both manifests have identical content")
			Expect(destContent).To(Equal(sourceContent))
			Expect(destContent).To(Equal(testFixtures.GetSimpleManifest()))

			By("Cleaning up destination tag")
			session = ExecuteKubectlMft("delete", "-t", destTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should copy manifest to different repository", func() {
			destTag := CreateUniqueTag("cp-test-different-repo")

			By("Copying manifest to different repository")
			session := ExecuteKubectlMft("cp", sourceTag, destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying destination manifest exists and matches source")
			session = ExecuteKubectlMft("dump", "-t", destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(Equal(testFixtures.GetSimpleManifest()))

			By("Cleaning up destination tag")
			session = ExecuteKubectlMft("delete", "-t", destTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should copy manifest to different registry", func() {
			// Parse source tag to get repository part
			destTag := CreateUniqueTag("cp-test-cross-registry")
			// Replace registry part to simulate cross-registry copy
			destTag = "localhost:5001/" + destTag[len("localhost:5000/"):]

			By("Copying manifest to different registry")
			session := ExecuteKubectlMft("cp", sourceTag, destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying destination manifest exists")
			session = ExecuteKubectlMft("dump", "-t", destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(Equal(testFixtures.GetSimpleManifest()))

			By("Cleaning up destination tag")
			session = ExecuteKubectlMft("delete", "-t", destTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Context("Error cases", func() {
		It("should fail when source tag does not exist", func() {
			nonExistentTag := CreateUniqueTag("non-existent")
			destTag := CreateUniqueTag("cp-test-dest")

			By("Attempting to copy from non-existent source")
			session := ExecuteKubectlMft("cp", nonExistentTag, destTag)
			Eventually(session).Should(gexec.Exit(1))

			By("Verifying error message")
			Expect(session.Err).To(gbytes.Say("source tag .* not found in local storage"))
		})

		It("should fail when destination tag already exists", func() {
			By("Creating destination tag with same content")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, "-t", sourceTag+"-existing")
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Attempting to copy to existing destination")
			session = ExecuteKubectlMft("cp", sourceTag, sourceTag+"-existing")
			Eventually(session).Should(gexec.Exit(1))

			By("Verifying error message")
			Expect(session.Err).To(gbytes.Say("destination tag .* already exists"))

			By("Cleaning up existing tag")
			session = ExecuteKubectlMft("delete", "-t", sourceTag+"-existing", "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should fail with non-existent simple source tag", func() {
			destTag := CreateUniqueTag("cp-test-dest")

			By("Attempting to copy with non-existent simple source tag")
			session := ExecuteKubectlMft("cp", "nonexistent-source", destTag)
			Eventually(session).Should(gexec.Exit(1))

			By("Verifying error message")
			Expect(session.Err).To(gbytes.Say("not found"))
		})

		It("should succeed with simple destination tag format", func() {
			By("Copying to simple destination tag")
			session := ExecuteKubectlMft("cp", sourceTag, "simple-dest:latest")
			Eventually(session).Should(gexec.Exit(0))

			By("Cleaning up simple destination tag")
			session = ExecuteKubectlMft("delete", "-t", "simple-dest:latest", "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should fail when insufficient arguments are provided", func() {
			By("Attempting to copy with only source tag")
			session := ExecuteKubectlMft("cp", sourceTag)
			Eventually(session).Should(gexec.Exit(1))

			By("Verifying error message")
			Expect(session.Err).To(gbytes.Say("accepts 2 arg"))
		})

		It("should fail when no arguments are provided", func() {
			By("Attempting to copy with no arguments")
			session := ExecuteKubectlMft("cp")
			Eventually(session).Should(gexec.Exit(1))

			By("Verifying error message")
			Expect(session.Err).To(gbytes.Say("accepts 2 arg"))
		})

		It("should fail when too many arguments are provided", func() {
			destTag := CreateUniqueTag("cp-test-dest")

			By("Attempting to copy with too many arguments")
			session := ExecuteKubectlMft("cp", sourceTag, destTag, "extra-arg")
			Eventually(session).Should(gexec.Exit(1))

			By("Verifying error message")
			Expect(session.Err).To(gbytes.Say("accepts 2 arg"))
		})
	})

	Context("Metadata preservation", func() {
		It("should preserve manifest content and structure", func() {
			destTag := CreateUniqueTag("cp-test-metadata")

			By("Copying manifest")
			session := ExecuteKubectlMft("cp", sourceTag, destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Dumping both source and destination")
			sourceSession := ExecuteKubectlMft("dump", "-t", sourceTag)
			Eventually(sourceSession, 10*time.Second).Should(gexec.Exit(0))
			sourceContent := string(sourceSession.Out.Contents())

			destSession := ExecuteKubectlMft("dump", "-t", destTag)
			Eventually(destSession, 10*time.Second).Should(gexec.Exit(0))
			destContent := string(destSession.Out.Contents())

			By("Verifying exact content match")
			Expect(destContent).To(Equal(sourceContent))
			Expect(destContent).To(ContainSubstring("kind: Deployment"))
			Expect(destContent).To(ContainSubstring("name: test-app"))

			By("Cleaning up destination tag")
			session = ExecuteKubectlMft("delete", "-t", destTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Context("List integration", func() {
		It("should show both source and destination in list output", func() {
			destTag := CreateUniqueTag("cp-test-list")

			By("Copying manifest")
			session := ExecuteKubectlMft("cp", sourceTag, destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Listing all manifests in JSON format")
			session = ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying both tags appear in list")
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("cp-test-source"))
			Expect(output).To(ContainSubstring("cp-test-list"))

			By("Cleaning up destination tag")
			session = ExecuteKubectlMft("delete", "-t", destTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Context("Silent success behavior", func() {
		It("should produce no output on successful copy", func() {
			destTag := CreateUniqueTag("cp-test-silent")

			By("Copying manifest")
			session := ExecuteKubectlMft("cp", sourceTag, destTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying no stdout output")
			Expect(session.Out.Contents()).To(BeEmpty())

			By("Verifying no stderr output")
			Expect(session.Err.Contents()).To(BeEmpty())

			By("Cleaning up destination tag")
			session = ExecuteKubectlMft("delete", "-t", destTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})
})
