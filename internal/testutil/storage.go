package testutil

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type FakeStorage struct {
	mu         sync.Mutex
	objects    map[string][]byte
	FailUpload bool
	FailDelete bool
}

func NewFakeStorage() *FakeStorage {
	return &FakeStorage{objects: make(map[string][]byte)}
}

func (s *FakeStorage) Upload(ctx context.Context, key string, data []byte) error {
	if s.FailUpload {
		return fmt.Errorf("fake storage: upload disabled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	stored := make([]byte, len(data))
	copy(stored, data)
	s.objects[key] = stored
	return nil
}

func (s *FakeStorage) Delete(ctx context.Context, key string) error {
	if s.FailDelete {
		return fmt.Errorf("fake storage: delete disabled")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

func (s *FakeStorage) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return "https://storage.test/" + key, nil
}

func (s *FakeStorage) Get(key string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, exists := s.objects[key]
	return data, exists
}

func (s *FakeStorage) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.objects)
}
