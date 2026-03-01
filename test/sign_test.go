// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Signing and Verification", func() {
	var manifestPath string

	BeforeEach(func() {
		manifestPath = testFixtures.CreateManifestFile("sign-test.yaml", testFixtures.GetSimpleManifest())
	})

	Describe("Full workflow: key generate → pack (auto-sign) → push → pull (auto-verify) → dump", func() {
		var testTag string

		BeforeEach(func() {
			testTag = fmt.Sprintf("%s/sign-workflow:%d",
				testRegistry.GetRegistryURL(), time.Now().UnixNano())
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should sign during pack and verify during pull", func() {
			By("Packing with auto-sign (key was generated in BeforeSuite)")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Pushing to registry (with signature)")
			session = ExecuteKubectlMft("push", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Deleting local copy")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Pulling from registry (with auto-verify)")
			session = ExecuteKubectlMft("pull", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Dumping to verify content")
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			originalContent, err := os.ReadFile(manifestPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(session.Out.Contents()).To(Equal(originalContent))
		})
	})

	Describe("Pack with --skip-sign", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("sign-skip-pack")
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should pack without signing when --skip-sign is used", func() {
			By("Packing with --skip-sign")
			session := ExecuteKubectlMft("pack", "--skip-sign", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Verifying content via dump")
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(Equal(testFixtures.GetSimpleManifest()))
		})
	})

	Describe("Pull with --skip-verify", func() {
		var testTag string

		BeforeEach(func() {
			testTag = fmt.Sprintf("%s/sign-skip-verify:%d",
				testRegistry.GetRegistryURL(), time.Now().UnixNano())
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should pull without verifying when --skip-verify is used", func() {
			By("Packing with --skip-sign")
			session := ExecuteKubectlMft("pack", "--skip-sign", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Pushing to registry")
			session = ExecuteKubectlMft("push", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Deleting local copy")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Pulling with --skip-verify")
			session = ExecuteKubectlMft("pull", "--skip-verify", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Verifying content via dump")
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Describe("Standalone sign and verify commands", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("sign-standalone")
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should sign and verify a packed manifest", func() {
			By("Packing without signing")
			session := ExecuteKubectlMft("pack", "--skip-sign", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Signing the manifest")
			session = ExecuteKubectlMft("sign", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Signed"))

			By("Verifying the signature")
			session = ExecuteKubectlMft("verify", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Verified"))
		})
	})

	Describe("Key import and verify", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("sign-import")
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should verify with imported public key", func() {
			By("Exporting the default public key")
			session := ExecuteKubectlMft("key", "export")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			pubKeyContent := session.Out.Contents()
			Expect(len(pubKeyContent)).To(BeNumerically(">", 0))

			By("Writing exported key to a temp file")
			tmpPubKey := testFixtures.CreateManifestFile("exported.pub", string(pubKeyContent))

			By("Importing the public key with a different name")
			session = ExecuteKubectlMft("key", "import", tmpPubKey, "--name", "imported")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Packing with auto-sign")
			session = ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Verifying with imported key")
			session = ExecuteKubectlMft("verify", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("Verified"))

			By("Cleaning up imported key")
			session = ExecuteKubectlMft("key", "delete", "imported")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Describe("Key list command", func() {
		It("should list the signing keys", func() {
			session := ExecuteKubectlMft("key", "list")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("NAME"))
			Expect(output).To(ContainSubstring("private"))
			Expect(output).To(ContainSubstring("default"))
		})
	})

	Describe("No key scenarios", func() {
		It("should fail to pack without signing key (with guidance message)", func() {
			emptyKeyDir, err := os.MkdirTemp("", "kubectl-mft-test-nokeys-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(emptyKeyDir)

			testTag := CreateUniqueTag("sign-nokey-pack")
			session := ExecuteKubectlMftWithKeyDir(emptyKeyDir, "pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("signing key.*not found"))
			Expect(session.Err).To(gbytes.Say("key generate"))

			// Clean up any partial data
			cleanSession := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(cleanSession, 10*time.Second).Should(gexec.Exit())
		})

		It("should succeed with --skip-sign when no key exists", func() {
			emptyKeyDir, err := os.MkdirTemp("", "kubectl-mft-test-nokeys-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(emptyKeyDir)

			testTag := CreateUniqueTag("sign-nokey-skip")
			session := ExecuteKubectlMftWithKeyDir(emptyKeyDir, "pack", "--skip-sign", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			cleanSession := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(cleanSession, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should fail to pull without verification key and delete pulled data", func() {
			emptyKeyDir, err := os.MkdirTemp("", "kubectl-mft-test-nokeys-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(emptyKeyDir)

			// Pack and push with signing
			testTag := fmt.Sprintf("%s/sign-nokey-pull:%d",
				testRegistry.GetRegistryURL(), time.Now().UnixNano())
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("push", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			// Pull with no verification keys → error
			session = ExecuteKubectlMftWithKeyDir(emptyKeyDir, "pull", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("no verification keys found"))
			Expect(session.Err).To(gbytes.Say("key import"))

			// Verify pulled data was deleted (dump should fail)
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(1))
		})

		It("should succeed with --skip-verify when no key exists", func() {
			emptyKeyDir, err := os.MkdirTemp("", "kubectl-mft-test-nokeys-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(emptyKeyDir)

			testTag := fmt.Sprintf("%s/sign-nokey-skipverify:%d",
				testRegistry.GetRegistryURL(), time.Now().UnixNano())
			session := ExecuteKubectlMft("pack", "--skip-sign", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("push", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMftWithKeyDir(emptyKeyDir, "pull", "--skip-verify", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			// Clean up
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Describe("Wrong key verification", func() {
		It("should fail to verify with wrong public key", func() {
			// Generate a different key pair in a separate key directory
			wrongKeyDir, err := os.MkdirTemp("", "kubectl-mft-test-wrongkeys-*")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(wrongKeyDir)

			session := ExecuteKubectlMftWithKeyDir(wrongKeyDir, "key", "generate")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			// Pack with auto-sign using the default test key
			testTag := CreateUniqueTag("sign-wrongkey")
			session = ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			// Verify with wrong key → should fail
			session = ExecuteKubectlMftWithKeyDir(wrongKeyDir, "verify", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(1))
			Expect(session.Err).To(gbytes.Say("verification failed"))

			// Clean up
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})
})
