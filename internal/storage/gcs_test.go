package storage_test

import (
	"auto-myself-api/internal/google"
	"auto-myself-api/internal/storage"
	"context"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
)

// TestGCSRoundTrip talks to the real bucket. It is skipped unless RUN_LIVE_GCS_TESTS is set and
// Application Default Credentials can access STORAGE_BUCKET.
func TestGCSRoundTrip(t *testing.T) {
	if os.Getenv("RUN_LIVE_GCS_TESTS") == "" {
		t.Skip("set RUN_LIVE_GCS_TESTS=1 to run against Cloud Storage")
	}

	ctx := context.Background()
	gcs := storage.NewGCS(google.GetBucketName())
	defer gcs.Close()

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
