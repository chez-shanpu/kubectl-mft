// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Schema Command", func() {
	var savedSchemaDir string

	BeforeEach(func() {
		savedSchemaDir = testSchemaDir

		tmpDir, err := os.MkdirTemp("", "kubectl-mft-schema-test-*")
		Expect(err).NotTo(HaveOccurred())
		testSchemaDir = tmpDir
	})

	AfterEach(func() {
		os.RemoveAll(testSchemaDir)
		testSchemaDir = savedSchemaDir
	})

	Context("Schema lifecycle (add, list, delete)", func() {
		It("should add, list, and delete CRD schema successfully", func() {
			crdPath := testFixtures.CreateManifestFile("crd.yaml", testFixtures.GetCRDManifest())

			By("Adding CRD schema")
			session := ExecuteKubectlMft("schema", "add", "-f", crdPath)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("registered successfully"))

			By("Listing registered schemas")
			session = ExecuteKubectlMft("schema", "list")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("example.com"))
			Expect(output).To(ContainSubstring("MyResource"))
			Expect(output).To(ContainSubstring("v1"))

			By("Deleting CRD schema")
			session = ExecuteKubectlMft("schema", "delete", "example.com/MyResource")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying schema is removed from list")
			session = ExecuteKubectlMft("schema", "list")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("No CRD schemas registered"))
		})
	})

	Context("Pack with CRD schema validation", func() {
		It("should validate custom resource against registered CRD schema", func() {
			crdPath := testFixtures.CreateManifestFile("crd-for-pack.yaml", testFixtures.GetCRDManifest())
			crPath := testFixtures.CreateManifestFile("cr.yaml", testFixtures.GetCustomResourceManifest())
			testTag := CreateUniqueTag("schema-pack")

			By("Registering CRD schema")
			session := ExecuteKubectlMft("schema", "add", "-f", crdPath)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Packing custom resource manifest")
			session = ExecuteKubectlMft("pack", "-f", crPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Cleaning up")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Context("Schema error cases", func() {
		It("should fail to delete non-existent schema", func() {
			By("Attempting to delete a non-existent schema")
			session := ExecuteKubectlMft("schema", "delete", "nonexistent.io/FakeResource")
			Eventually(session, 10*time.Second).Should(gexec.Exit(1))
		})

		It("should list empty when no schemas registered", func() {
			By("Listing schemas with none registered")
			session := ExecuteKubectlMft("schema", "list")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("No CRD schemas registered"))
		})
	})
})
