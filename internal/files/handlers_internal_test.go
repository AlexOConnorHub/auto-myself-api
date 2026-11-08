package files

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type blockingReadCloser struct {
	release chan struct{}
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{release: make(chan struct{})}
}

func (b *blockingReadCloser) Read([]byte) (int, error) {
	<-b.release
	return 0, io.EOF
}

// Close deliberately does nothing: net/http's server body Close cannot interrupt an in-flight
// Read, so readUploadBody must not depend on it.
func (b *blockingReadCloser) Close() error { return nil }

func TestReadUploadBodyReturnsOnContextCancel(t *testing.T) {
	body := newBlockingReadCloser()
	defer close(body.release)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := readUploadBody(ctx, body)
		done <- err
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("readUploadBody did not return after context cancellation")
	}
}
