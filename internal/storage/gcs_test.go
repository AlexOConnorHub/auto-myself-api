package storage_test

import (
	"auto-myself-api/internal/google"
	"auto-myself-api/internal/storage"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

func TestGCSRoundTrip(t *testing.T) {
	if os.Getenv("RUN_LIVE_GCS_TESTS") == "" {
		t.Skip("set RUN_LIVE_GCS_TESTS=1 to run against Cloud Storage")
	}

	ctx := context.Background()
	gcs := storage.NewGCS(google.GetBucketName())
	defer gcs.Close()

	if err := gcs.VerifySignedURL(ctx); err != nil {
		t.Fatalf("verify signed-URL signing: %v", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("allocating key: %v", err)
	}
	key := "test/" + id.String()
	data := []byte("live storage round trip")

	if err := gcs.Upload(ctx, key, data); err != nil {
		t.Fatalf("upload: %v", err)
	}
	defer func() {
		if err := gcs.Delete(ctx, key); err != nil {
			t.Errorf("cleanup delete: %v", err)
		}
	}()

	url, err := gcs.SignedURL(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("signed URL: %v", err)
	}
	if url == "" {
		t.Fatal("expected a signed URL")
	}
}

func TestGCSVerifySignedURLChecksContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := storage.NewGCS("unused-bucket").VerifySignedURL(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("VerifySignedURL() error = %v, want context.Canceled", err)
	}
}
