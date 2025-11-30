// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Integration Tests", func() {

	var manifestPath string

	BeforeEach(func() {
		manifestPath = testFixtures.CreateManifestFile("test-deployment.yaml", testFixtures.GetSimpleManifest())
	})

	Describe("Full workflow: Pack → List → Dump → Delete", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("workflow")
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should complete full lifecycle successfully", func() {
			By("Packing a manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Verifying it appears in list")
			session = ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			var listResult []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &listResult)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(listResult)).To(BeNumerically(">", 0))

			By("Dumping the manifest")
			outputPath := filepath.Join(testFixtures.GetTempDir(), "dumped.yaml")
			session = ExecuteKubectlMft("dump", testTag, "-o", outputPath)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			Expect(outputPath).To(BeAnExistingFile())

			By("Verifying dumped content matches original")
			originalContent, err := os.ReadFile(manifestPath)
			Expect(err).NotTo(HaveOccurred())
			dumpedContent, err := os.ReadFile(outputPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(dumpedContent).To(Equal(originalContent))

			By("Deleting the manifest")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying it no longer appears in list")
			session = ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			var afterDeleteList []map[string]interface{}
			err = json.Unmarshal(session.Out.Contents(), &afterDeleteList)
			Expect(err).NotTo(HaveOccurred())

			// Should not find the deleted tag
			for _, m := range afterDeleteList {
				tagParts := strings.Split(testTag, ":")
				if len(tagParts) == 2 && m["tag"].(string) == tagParts[1] {
					Fail("Deleted tag should not appear in list")
				}
			}
		})
	})

	Describe("Full workflow: Pack → Push → Pull → Dump", func() {
		var testTag string

		BeforeEach(func() {
			testTag = fmt.Sprintf("%s/push-pull-test:%d",
				testRegistry.GetRegistryURL(), time.Now().UnixNano())
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should push and pull successfully", func() {
			By("Packing a manifest")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Pushing to registry")
			session = ExecuteKubectlMft("push", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Deleting local copy")
			session = ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Pulling from registry")
			session = ExecuteKubectlMft("pull", testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))

			By("Dumping pulled manifest")
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying content is correct")
			originalContent, err := os.ReadFile(manifestPath)
			Expect(err).NotTo(HaveOccurred())
			pulledContent := session.Out.Contents()
			Expect(pulledContent).To(Equal(originalContent))
		})
	})

	Describe("Multiple tags in same repository workflow", func() {
		var baseRepo string
		var tags []string

		BeforeEach(func() {
			baseRepo = fmt.Sprintf("%s/multi-tag-workflow", testRegistry.GetRegistryURL())
			tags = []string{
				baseRepo + ":v1.0.0",
				baseRepo + ":v2.0.0",
				baseRepo + ":v3.0.0",
			}
		})

		AfterEach(func() {
			// Cleanup all tags
			for _, tag := range tags {
				session := ExecuteKubectlMft("delete", tag, "--force")
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			}
		})

		It("should manage multiple tags correctly", func() {
			By("Packing three different tags")
			for _, tag := range tags {
				session := ExecuteKubectlMft("pack", "-f", manifestPath, tag)
				Eventually(session, 30*time.Second).Should(gexec.Exit(0))
				time.Sleep(100 * time.Millisecond)
			}

			By("Verifying all tags appear in list")
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			var listResult []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &listResult)
			Expect(err).NotTo(HaveOccurred())

			foundCount := 0
			for _, m := range listResult {
				if strings.Contains(m["repository"].(string), "multi-tag-workflow") {
					foundCount++
				}
			}
			Expect(foundCount).To(Equal(3))

			By("Deleting middle tag")
			session = ExecuteKubectlMft("delete", tags[1], "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying other tags still exist")
			session = ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			err = json.Unmarshal(session.Out.Contents(), &listResult)
			Expect(err).NotTo(HaveOccurred())

			foundTags := make(map[string]bool)
			for _, m := range listResult {
				if strings.Contains(m["repository"].(string), "multi-tag-workflow") {
					foundTags[m["tag"].(string)] = true
				}
			}

			Expect(foundTags["v1.0.0"]).To(BeTrue())
			Expect(foundTags["v2.0.0"]).To(BeFalse())
			Expect(foundTags["v3.0.0"]).To(BeTrue())

			By("Verifying remaining tags can be dumped")
			session = ExecuteKubectlMft("dump", tags[0])
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("dump", tags[2])
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should share index.json for all tags", func() {
			By("Packing multiple tags")
			for _, tag := range tags {
				session := ExecuteKubectlMft("pack", "-f", manifestPath, tag)
				Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			}

			By("Checking index.json exists")
			repoDir := filepath.Join(testStorageDir, baseRepo)
			indexPath := filepath.Join(repoDir, "index.json")
			Expect(indexPath).To(BeAnExistingFile())

			By("Verifying index.json contains all tags")
			indexContent, err := os.ReadFile(indexPath)
			Expect(err).NotTo(HaveOccurred())

			var index struct {
				Manifests []struct {
					Annotations map[string]string `json:"annotations"`
				} `json:"manifests"`
			}
			err = json.Unmarshal(indexContent, &index)
			Expect(err).NotTo(HaveOccurred())
			Expect(index.Manifests).To(HaveLen(3))

			foundTags := make(map[string]bool)
			for _, m := range index.Manifests {
				tag := m.Annotations["org.opencontainers.image.ref.name"]
				foundTags[tag] = true
			}

			Expect(foundTags["v1.0.0"]).To(BeTrue())
			Expect(foundTags["v2.0.0"]).To(BeTrue())
			Expect(foundTags["v3.0.0"]).To(BeTrue())
		})
	})

	Describe("Path and Dump integration", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("path-dump")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should access manifest via path command", func() {
			By("Getting blob path")
			session := ExecuteKubectlMft("path", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			blobPath := strings.TrimSpace(string(session.Out.Contents()))

			By("Reading content from path")
			pathContent, err := os.ReadFile(blobPath)
			Expect(err).NotTo(HaveOccurred())

			By("Getting content via dump")
			session = ExecuteKubectlMft("dump", testTag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			dumpContent := session.Out.Contents()

			By("Verifying content matches")
			Expect(pathContent).To(Equal(dumpContent))
		})
	})

	Describe("Multiple repositories workflow", func() {
		var repo1Tag, repo2Tag, repo3Tag string

		BeforeEach(func() {
			repo1Tag = CreateUniqueTag("repo-a")
			repo2Tag = CreateUniqueTag("repo-b")
			repo3Tag = CreateUniqueTag("repo-c")
		})

		AfterEach(func() {
			for _, tag := range []string{repo1Tag, repo2Tag, repo3Tag} {
				session := ExecuteKubectlMft("delete", tag, "--force")
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			}
		})

		It("should manage multiple repositories independently", func() {
			By("Packing to different repositories")
			for _, tag := range []string{repo1Tag, repo2Tag, repo3Tag} {
				session := ExecuteKubectlMft("pack", "-f", manifestPath, tag)
				Eventually(session, 30*time.Second).Should(gexec.Exit(0))
			}

			By("Verifying all repositories in list")
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			var listResult []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &listResult)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(listResult)).To(BeNumerically(">=", 3))

			By("Deleting from one repository")
			session = ExecuteKubectlMft("delete", repo2Tag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			By("Verifying other repositories unaffected")
			session = ExecuteKubectlMft("dump", repo1Tag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			session = ExecuteKubectlMft("dump", repo3Tag)
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})
	})
})
