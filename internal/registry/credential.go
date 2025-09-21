package registry

import (
	"fmt"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

func newCredentialFunc() (auth.CredentialFunc, error) {
	s, err := newCredentialStore()
	if err != nil {
		return nil, err
	}
	return credentials.Credential(s), nil
}

// newCredentialStore creates a credential store with secure defaults
func newCredentialStore() (credentials.Store, error) {
	opt := credentials.StoreOptions{
		AllowPlaintextPut: false, // Secure default
	}
	s, err := credentials.NewStoreFromDocker(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential store: %w", err)
	}
	return s, nil
}
