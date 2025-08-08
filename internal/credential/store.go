// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of kubectl-mft

package credential

import (
	"fmt"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

func CreateFunc() (auth.CredentialFunc, error) {
	s, err := CreateStore()
	if err != nil {
		return nil, err
	}
	return credentials.Credential(s), nil
}

// CreateStore creates a credential store with secure defaults
func CreateStore() (credentials.Store, error) {
	opt := credentials.StoreOptions{
		AllowPlaintextPut: false, // Secure default
	}
	s, err := credentials.NewStoreFromDocker(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential store: %w", err)
	}
	return s, nil
}
