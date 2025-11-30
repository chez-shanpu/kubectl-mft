// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

//go:build e2e

package test

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"gopkg.in/yaml.v3"
)

var _ = Describe("List Command", func() {

	var manifestPath string

	BeforeEach(func() {
		manifestPath = testFixtures.CreateManifestFile("test-deployment.yaml", testFixtures.GetSimpleManifest())
	})

	Context("when listing empty repository", func() {
		It("should show no manifests with table format", func() {
			session := ExecuteKubectlMft("list")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should return empty array with JSON format", func() {
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should return empty array with YAML format", func() {
			session := ExecuteKubectlMft("list", "-o", "yaml")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []interface{}
			err := yaml.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Context("when listing single manifest", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("list-single")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should show the manifest in table format", func() {
			session := ExecuteKubectlMft("list")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			output := string(session.Out.Contents())
			Expect(output).To(ContainSubstring("REPOSITORY"))
			Expect(output).To(ContainSubstring("TAG"))
			Expect(output).To(ContainSubstring("SIZE"))
			Expect(output).To(ContainSubstring("CREATED"))
		})

		It("should return valid JSON with all fields", func() {
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))

			manifest := result[0]
			Expect(manifest).To(HaveKey("repository"))
			Expect(manifest).To(HaveKey("tag"))
			Expect(manifest).To(HaveKey("size"))
			Expect(manifest).To(HaveKey("created"))
		})

		It("should return valid YAML with all fields", func() {
			session := ExecuteKubectlMft("list", "-o", "yaml")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := yaml.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))

			manifest := result[0]
			Expect(manifest).To(HaveKey("repository"))
			Expect(manifest).To(HaveKey("tag"))
			Expect(manifest).To(HaveKey("size"))
			Expect(manifest).To(HaveKey("created"))
		})
	})

	Context("when listing multiple manifests from same repository", func() {
		var baseRepo string
		var tag1, tag2, tag3 string

		BeforeEach(func() {
			baseRepo = "localhost:5000/multi-tag-test"
			tag1 = baseRepo + ":v1.0.0"
			tag2 = baseRepo + ":v2.0.0"
			tag3 = baseRepo + ":v3.0.0"

			// Pack three different tags to the same repository
			for _, tag := range []string{tag1, tag2, tag3} {
				session := ExecuteKubectlMft("pack", "-f", manifestPath, tag)
				Eventually(session, 30*time.Second).Should(gexec.Exit(0))
				time.Sleep(100 * time.Millisecond) // Small delay to ensure different timestamps
			}
		})

		AfterEach(func() {
			for _, tag := range []string{tag1, tag2, tag3} {
				session := ExecuteKubectlMft("delete", tag, "--force")
				Eventually(session, 10*time.Second).Should(gexec.Exit(0))
			}
		})

		It("should list all tags from the same repository", func() {
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(result)).To(BeNumerically(">=", 3))

			// Count how many are from our test repository
			count := 0
			for _, m := range result {
				if strings.Contains(m["repository"].(string), "multi-tag-test") {
					count++
				}
			}
			Expect(count).To(Equal(3))
		})

		It("should sort tags correctly", func() {
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())

			// Extract tags from our test repository
			var tags []string
			for _, m := range result {
				if strings.Contains(m["repository"].(string), "multi-tag-test") {
					tags = append(tags, m["tag"].(string))
				}
			}

			// Tags should be sorted alphabetically
			Expect(tags).To(Equal([]string{"v1.0.0", "v2.0.0", "v3.0.0"}))
		})
	})

	Context("when listing multiple manifests from different repositories", func() {
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

		It("should list manifests from all repositories", func() {
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())

			// Should have at least our two manifests
			Expect(len(result)).To(BeNumerically(">=", 2))

			foundRepoA := false
			foundRepoB := false
			for _, m := range result {
				repo := m["repository"].(string)
				if strings.Contains(repo, "repo-a") {
					foundRepoA = true
				}
				if strings.Contains(repo, "repo-b") {
					foundRepoB = true
				}
			}

			Expect(foundRepoA).To(BeTrue())
			Expect(foundRepoB).To(BeTrue())
		})

		It("should sort by repository name first", func() {
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())

			// Find our two repositories
			var repoAIndex, repoBIndex int
			for i, m := range result {
				repo := m["repository"].(string)
				if strings.Contains(repo, "repo-a") {
					repoAIndex = i
				}
				if strings.Contains(repo, "repo-b") {
					repoBIndex = i
				}
			}

			// repo-a should come before repo-b
			Expect(repoAIndex).To(BeNumerically("<", repoBIndex))
		})
	})

	Context("when using invalid output format", func() {
		It("should fail with appropriate error message", func() {
			session := ExecuteKubectlMft("list", "-o", "invalid-format")
			Eventually(session).Should(gexec.Exit(1))
		})
	})

	Context("when verifying size formatting", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("size-test")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should display human-readable size", func() {
			session := ExecuteKubectlMft("list")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			output := string(session.Out.Contents())
			// Size should be in human-readable format (B, KB, MB, etc.)
			Expect(output).To(MatchRegexp(`\d+(\.\d+)?[KMGTPE]?B`))
		})
	})

	Context("when verifying created timestamp", func() {
		var testTag string

		BeforeEach(func() {
			testTag = CreateUniqueTag("timestamp-test")
			session := ExecuteKubectlMft("pack", "-f", manifestPath, testTag)
			Eventually(session, 30*time.Second).Should(gexec.Exit(0))
		})

		AfterEach(func() {
			session := ExecuteKubectlMft("delete", testTag, "--force")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))
		})

		It("should include created field in JSON format", func() {
			session := ExecuteKubectlMft("list", "-o", "json")
			Eventually(session, 10*time.Second).Should(gexec.Exit(0))

			var result []map[string]interface{}
			err := json.Unmarshal(session.Out.Contents(), &result)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeEmpty())

			for _, m := range result {
				Expect(m).To(HaveKey("created"))
				Expect(m["created"]).NotTo(BeEmpty())
			}
		})
	})
})
