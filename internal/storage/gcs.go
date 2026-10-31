package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

// GCS implements files.StorageRepository against a single Cloud Storage bucket. The client is
// created lazily so that a debug-mode process can start without Google credentials and reused
// for every operation afterwards.
type GCS struct {
	bucketName string
	once       sync.Once
	client     *storage.Client
	err        error
}

func NewGCS(bucketName string) *GCS {
	return &GCS{bucketName: bucketName}
}

func (g *GCS) ensureClient() (*storage.Client, error) {
	g.once.Do(func() {
		g.client, g.err = storage.NewClient(context.Background())
	})
	return g.client, g.err
}

func (g *GCS) Upload(ctx context.Context, key string, data []byte) error {
	client, err := g.ensureClient()
	if err != nil {
		return fmt.Errorf("creating storage client: %w", err)
	}

	writer := client.Bucket(g.bucketName).Object(key).NewWriter(ctx)
	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return fmt.Errorf("writing object %s: %w", key, err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing object %s: %w", key, err)
	}
	return nil
}

func (g *GCS) Delete(ctx context.Context, key string) error {
	client, err := g.ensureClient()
	if err != nil {
		return fmt.Errorf("creating storage client: %w", err)
	}
	if err := client.Bucket(g.bucketName).Object(key).Delete(ctx); err != nil {
		return fmt.Errorf("deleting object %s: %w", key, err)
	}
	return nil
}

func (g *GCS) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	client, err := g.ensureClient()
	if err != nil {
		return "", fmt.Errorf("creating storage client: %w", err)
	}
	url, err := client.Bucket(g.bucketName).SignedURL(key, &storage.SignedURLOptions{
		Expires: time.Now().Add(ttl),
		Method:  "GET",
	})
	if err != nil {
		return "", fmt.Errorf("signing URL for object %s: %w", key, err)
	}
	return url, nil
}

func (g *GCS) Close() error {
	if g.client == nil {
		return nil
	}
	return g.client.Close()
}
