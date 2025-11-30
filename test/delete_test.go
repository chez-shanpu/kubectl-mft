// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Delete Command", func() {

	var manifestPath string

	BeforeEach(func() {
		manifestPath = testFixtures.CreateManifestFile("test-deployment.yaml", testFixtures.GetSimpleManifest())
	})

	Context("when deleting a single tag", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("delete-single")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		It("should successfully delete the manifest", func() {
			By("Deleting the manifest")
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying output contains deletion information")
			Expect(session.Out).To(gbytes.Say("Deleted"))

			By("Verifying the manifest is no longer listed")
			session = ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())

			// Should not find our deleted tag
			for _, m := range result {
				if strings.Contains(testTag, m["tag"].(string)) {
					Fail("Deleted tag should not appear in list")
				}
			}
		})

		It("should show deletion confirmation in output", func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			output := string(session.Out.Contents())
			// Should contain "Deleted" message
			Expect(output).To(ContainSubstring("Deleted"))
			Expect(output).To(ContainSubstring("localhost:5000/delete-single"))
		})
	})

	Context("when deleting and verifying blobs are removed", func() {
		var testTag string
		var repoDir string

		BeforeEach(func() {
			testTag = "localhost:5000/blob-cleanup-test:v1.0.0"
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			repoDir = filepath.Join(testStorageDir, "localhost:5000", "blob-cleanup-test")
		})

		It("should remove orphaned blobs", func() {
			By("Checking blobs directory exists before deletion")
			blobsDir := filepath.Join(repoDir, "blobs")
			Expect(blobsDir).To(BeADirectory())

			By("Counting blobs before deletion")
			initialBlobCount := countBlobs(blobsDir)
			Expect(initialBlobCount).To(BeNumerically(">", 0))

			By("Deleting the manifest")
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying repository directory is removed (last tag)")
			Expect(repoDir).NotTo(BeADirectory())
		})
	})

	Context("when deleting one tag while keeping others in same repository", func() {
		var baseRepo string
		var tag1, tag2 string
		var repoDir string

		BeforeEach(func() {
			baseRepo = "localhost:5000/multi-tag-delete"
			tag1 = baseRepo + ":v1.0.0"
			tag2 = baseRepo + ":v2.0.0"
			repoDir = filepath.Join(testStorageDir, "localhost:5000", "multi-tag-delete")

			// Pack the same manifest twice (will share blobs)
			session := ExecuteKubectlMft("pack", "-f", manifestPath, tag1)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("pack", "-f", manifestPath, tag2)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			for _, tag := range []string{tag1, tag2} {
				session := ExecuteKubectlMft("delete", tag, "--force")
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			}
		})

		It("should keep other tags when deleting one tag", func() {
			By("Deleting only tag1")
			session := ExecuteKubectlMft("delete", tag1, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying tag2 still exists")
			session = ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())

			foundTag2 := false
			foundTag1 := false
			for _, m := range result {
				tag := m["tag"].(string)
				if tag == "v2.0.0" && strings.Contains(m["repository"].(string), "multi-tag-delete") {
					foundTag2 = true
				}
				if tag == "v1.0.0" && strings.Contains(m["repository"].(string), "multi-tag-delete") {
					foundTag1 = true
				}
			}

			Expect(foundTag2).To(BeTrue(), "tag2 should still exist")
			Expect(foundTag1).To(BeFalse(), "tag1 should be deleted")
		})

		It("should not remove shared blobs when one tag is deleted", func() {
			blobsDir := filepath.Join(repoDir, "blobs")

			By("Counting blobs before deletion")
			initialBlobCount := countBlobs(blobsDir)
			Expect(initialBlobCount).To(BeNumerically(">", 0))

			By("Deleting tag1")
			session := ExecuteKubectlMft("delete", tag1, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying some blobs still exist (shared with tag2)")
			remainingBlobCount := countBlobs(blobsDir)
			Expect(remainingBlobCount).To(BeNumerically(">", 0))

			By("Verifying tag2 can still be dumped")
			session = ExecuteKubectlMft("dump", tag2)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should remove repository directory when deleting last tag", func() {
			By("Deleting tag1")
			session := ExecuteKubectlMft("delete", tag1, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying repository still exists")
			Expect(repoDir).To(BeADirectory())

			By("Deleting tag2 (last tag)")
			session = ExecuteKubectlMft("delete", tag2, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying repository directory is removed")
			Expect(repoDir).NotTo(BeADirectory())
		})
	})

	Context("when deleting non-existent tag (idempotency)", func() {
		It("should succeed without error", func() {
			nonExistentTag := CreateUniqueTag("non-existent")
			session := ExecuteKubectlMft("delete", nonExistentTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should succeed when deleting same tag twice", func() {
			testTag := CreateUniqueTag("delete-twice")

			By("Packing a manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Deleting first time")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Deleting second time (should be idempotent)")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})

	Context("when simple tag does not exist", func() {
		It("should show not found warning", func() {
			session := ExecuteKubectlMft("delete", "nonexistent-simple-tag", "--force")
			Eventually(session).Should(gexec.Exit(0))
			Expect(session).To(gbytes.Say("not found"))
		})
	})

	Context("when tag argument is missing", func() {
		It("should fail with appropriate error message", func() {
			session := ExecuteKubectlMft("delete")
			Eventually(session).Should(gexec.Exit(1))
		})
	})

})

// Helper function to count blobs in a directory
func countBlobs(blobsDir string) int {
	count := 0
	filepath.Walk(blobsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}
