package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeSignedURLVerifier struct {
	calls int
	err   error
}

func (f *fakeSignedURLVerifier) VerifySignedURL(context.Context) error {
	f.calls++
	return f.err
}

func TestVerifyStorageSigning(t *testing.T) {
	sentinel := errors.New("signing unavailable")

	tests := []struct {
		name      string
		mode      string
		verifier  *fakeSignedURLVerifier
		wantCalls int
		wantErr   bool
	}{
		{name: "release success", mode: gin.ReleaseMode, verifier: &fakeSignedURLVerifier{}, wantCalls: 1},
		{name: "release failure", mode: gin.ReleaseMode, verifier: &fakeSignedURLVerifier{err: sentinel}, wantCalls: 1, wantErr: true},
		{name: "debug skips", mode: gin.DebugMode, verifier: &fakeSignedURLVerifier{err: sentinel}, wantCalls: 0},
		{name: "test skips", mode: gin.TestMode, verifier: &fakeSignedURLVerifier{err: sentinel}, wantCalls: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := verifyStorageSigning(context.Background(), test.mode, test.verifier)

			switch {
			case test.wantErr:
				if !errors.Is(err, sentinel) {
					t.Fatalf("error = %v, want the wrapped sentinel", err)
				}
				if !strings.Contains(err.Error(), "verifying storage signing") {
					t.Errorf("error = %q, want the storage-signing context", err)
				}
			case err != nil:
				t.Fatalf("unexpected error: %v", err)
			}

			if test.verifier.calls != test.wantCalls {
				t.Errorf("verifier calls = %d, want %d", test.verifier.calls, test.wantCalls)
			}
		})
	}
}
