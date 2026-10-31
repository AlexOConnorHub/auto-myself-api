package testutil

import (
	"auto-myself-api/internal/auth"
	"context"
	"sync"
)

type FakeIdentityVerifier struct {
	mu         sync.Mutex
	provider   string
	identities map[string]auth.VerifiedIdentity
	err        error
	lastNonce  string
}

func NewFakeIdentityVerifier(provider string) *FakeIdentityVerifier {
	return &FakeIdentityVerifier{
		provider:   provider,
		identities: make(map[string]auth.VerifiedIdentity),
	}
}

func (f *FakeIdentityVerifier) Register(token string, identity auth.VerifiedIdentity) {
	f.mu.Lock()
	defer f.mu.Unlock()

	identity.Provider = f.provider
	f.identities[token] = identity
}

func (f *FakeIdentityVerifier) SetError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *FakeIdentityVerifier) LastNonce() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastNonce
}

func (f *FakeIdentityVerifier) Verify(_ context.Context, rawToken, nonce string) (*auth.VerifiedIdentity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.lastNonce = nonce
	if f.err != nil {
		return nil, f.err
	}

	identity, exists := f.identities[rawToken]
	if !exists {
		return nil, auth.ErrInvalidIdentityToken
	}
	return &identity, nil
}
