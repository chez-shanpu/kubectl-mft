// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Pack Command", func() {
	Context("Debug container profile (YAML without apiVersion/kind)", func() {
		var manifestPath string
		var testTag string

		BeforeEach(func() {
			manifestPath = testFixtures.CreateManifestFile("debug-profile.yaml", testFixtures.GetDebugProfileManifest())
			testTag = CreateUniqueTag("pack-debug")
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should pack successfully with warning on stderr", func() {
			By("Packing debug profile manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Verifying warning on stderr about missing kind")
			Expect(session.Err).To(gbytes.Say("warning:.*missing 'kind' key"))

			By("Verifying content via dump")
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(Equal(testFixtures.GetDebugProfileManifest()))
		})

		It("should pack without warning when --skip-validation is used", func() {
			By("Packing with --skip-validation")
			session := ExecuteKubectlMft("pack", "--skip-validation", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Verifying no warning on stderr")
			Expect(session.Err.Contents()).To(BeEmpty())
		})
	})

	Context("Invalid manifest", func() {
		var manifestPath string

		BeforeEach(func() {
			manifestPath = testFixtures.CreateManifestFile("invalid.yaml", testFixtures.GetInvalidManifest())
		})

		It("should fail to pack an invalid manifest", func() {
			testTag := CreateUniqueTag("pack-invalid")

			By("Attempting to pack invalid manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(1))

			By("Verifying validation error on stderr")
			Expect(session.Err).To(gbytes.Say("validation failed"))
		})

		It("should pack an invalid manifest when --skip-validation is used", func() {
			testTag := CreateUniqueTag("pack-invalid-skip")

			By("Packing with --skip-validation")
			session := ExecuteKubectlMft("pack", "--skip-validation", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Cleaning up")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Context("Complex manifest (multi-document)", func() {
		var manifestPath string
		var testTag string

		BeforeEach(func() {
			manifestPath = testFixtures.CreateManifestFile("complex.yaml", testFixtures.GetComplexManifest())
			testTag = CreateUniqueTag("pack-complex")
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should pack multi-document manifest successfully", func() {
			By("Packing complex manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Verifying content via dump")
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(string(session.Out.Contents())).To(Equal(testFixtures.GetComplexManifest()))
		})
	})
})
