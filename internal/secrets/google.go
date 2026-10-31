package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"

	"auto-myself-api/internal/google"
	"auto-myself-api/internal/messages"
)

var ErrSecretNotFound = errors.New("secret not found")

type SecretValue struct {
	Value       string
	RetrievedAt time.Time
}

type SecretStore struct {
	secrets            map[string]*atomic.Pointer[SecretValue]
	watchers           map[string]map[int64]func(SecretValue)
	watchersMu         sync.Mutex
	watcherIDGenerator atomic.Int64
	clientSecret       *secretmanager.Client
}

func newSecretStore() *SecretStore {
	return &SecretStore{
		secrets:  make(map[string]*atomic.Pointer[SecretValue]),
		watchers: make(map[string]map[int64]func(SecretValue)),
	}
}

func NewEnvStore(required []string) (*SecretStore, error) {
	store := newSecretStore()
	for _, name := range required {
		value := os.Getenv(name)
		if value == "" {
			return nil, fmt.Errorf("required environment secret %s: %w", name, ErrSecretNotFound)
		}
		store.set(name, value)
	}
	return store, nil
}

func SetupSecrets(ctx context.Context, required []string) (*SecretStore, error) {
	if gin.Mode() == gin.DebugMode || gin.Mode() == gin.TestMode {
		return NewEnvStore(required)
	}

	store := newSecretStore()

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating secret manager client: %w", err)
	}
	store.clientSecret = client

	for _, name := range required {
		value, err := store.retrieveSecret(ctx, name)
		if err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("retrieving secret %s: %w", name, err)
		}
		store.set(name, value)
	}

	go store.subscribeLoop(ctx)

	return store, nil
}

func (s *SecretStore) set(name, value string) {
	valuePtr := &atomic.Pointer[SecretValue]{}
	valuePtr.Store(&SecretValue{Value: value, RetrievedAt: time.Now()})
	s.secrets[name] = valuePtr
}

func (s *SecretStore) Get(name string) (*SecretValue, error) {
	if value := os.Getenv(name); value != "" {
		return &SecretValue{Value: value, RetrievedAt: time.Now()}, nil
	}
	ptr, exists := s.secrets[name]
	if !exists {
		return nil, fmt.Errorf("secret %s: %w", name, ErrSecretNotFound)
	}
	value := ptr.Load()
	if value == nil {
		return nil, fmt.Errorf("secret %s: %w", name, ErrSecretNotFound)
	}
	return value, nil
}

func (s *SecretStore) MustGet(name string) *SecretValue {
	value, err := s.Get(name)
	if err != nil {
		panic(err)
	}
	return value
}

func (s *SecretStore) Watch(name string, handler func(SecretValue)) (cancel func()) {
	id := s.watcherIDGenerator.Add(1)

	s.watchersMu.Lock()
	if s.watchers[name] == nil {
		s.watchers[name] = make(map[int64]func(SecretValue))
	}
	s.watchers[name][id] = handler
	s.watchersMu.Unlock()

	return func() {
		s.watchersMu.Lock()
		defer s.watchersMu.Unlock()
		delete(s.watchers[name], id)
		if len(s.watchers[name]) == 0 {
			delete(s.watchers, name)
		}
	}
}

func (s *SecretStore) Close() error {
	if s.clientSecret == nil {
		return nil
	}
	return s.clientSecret.Close()
}

type GoogleSecretUpdate struct {
	EventType  string `json:"eventType"`
	DataFormat string `json:"dataFormat"`
	SecretId   string `json:"secretId"`
	Timestamp  string `json:"timestamp"`
}

func (s *SecretStore) listener(ctx context.Context, msg messages.UpdateMessage) error {
	var update GoogleSecretUpdate
	if err := json.Unmarshal(msg.Data, &update); err != nil {
		return fmt.Errorf("decoding secret update: %w", err)
	}

	parts := strings.Split(update.SecretId, "/")
	if len(parts) != 4 {
		return fmt.Errorf("secret ID %s is not in format `projects/{project}/secrets/{secret}`", update.SecretId)
	}
	secretName := parts[3]

	valuePtr, exists := s.secrets[secretName]
	if !exists {
		return nil
	}

	value, err := s.retrieveSecret(ctx, secretName)
	if err != nil {
		return fmt.Errorf("retrieving rotated secret %s: %w", secretName, err)
	}

	rotated := SecretValue{Value: value, RetrievedAt: time.Now()}
	valuePtr.Store(&rotated)

	s.watchersMu.Lock()
	handlers := make([]func(SecretValue), 0, len(s.watchers[secretName]))
	for _, handler := range s.watchers[secretName] {
		handlers = append(handlers, handler)
	}
	s.watchersMu.Unlock()

	for _, handler := range handlers {
		go handler(rotated)
	}

	return nil
}

func (s *SecretStore) subscribeLoop(ctx context.Context) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}

		err := messages.SubscribeUpdatesChannel(ctx, "API_SECRET_CHANGE", s.listener)
		if ctx.Err() != nil {
			return
		}
		slog.Error("secret update subscription stopped; retrying", "error", err, "retry_in", backoff)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < time.Minute {
			backoff *= 2
		}
	}
}

func (s *SecretStore) retrieveSecret(ctx context.Context, name string) (string, error) {
	if s.clientSecret == nil {
		return "", fmt.Errorf("secret %s: %w", name, ErrSecretNotFound)
	}
	projectID := google.GetNumericProjectID()
	result, err := s.clientSecret.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name),
	})
	if err != nil {
		return "", err
	}
	return string(result.Payload.Data), nil
}
