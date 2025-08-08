// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package credential

import (
	"strings"
	"testing"
)

// Test createCredentialStore function (error cases)
func TestCreateCredentialStore_ErrorHandling(t *testing.T) {
	// This test checks that the function handles Docker credential store errors gracefully
	// Note: This may pass or fail depending on the Docker setup in the test environment

	t.Run("credential store creation", func(t *testing.T) {
		store, err := CreateStore()

		// We can't predict if Docker credentials are available in test environment
		// So we just check that the error (if any) is properly formatted
		if err != nil {
			if !strings.Contains(err.Error(), "failed to create credential store") {
				t.Errorf("createCredentialStore() error = %q, expected to contain 'failed to create credential store'", err.Error())
			}
		} else {
			// If successful, store should not be nil
			if store == nil {
				t.Errorf("createCredentialStore() returned nil store with no error")
			}
		}
	})
}
