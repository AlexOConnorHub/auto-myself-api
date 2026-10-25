package secrets

import (
	"auto-myself-api/internal/google"
	"auto-myself-api/internal/messages"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pubsub "cloud.google.com/go/pubsub/v2"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

type SecretValue struct {
	Value       string
	RetrievedAt time.Time
}

type SecretWatcher struct {
	ID      *int64
	Handler func(SecretValue)
}

type SecretStore struct {
	secrets            map[string]*atomic.Pointer[SecretValue]
	watchers           map[string][]SecretWatcher
	watchersMu         sync.Mutex
	watcherIdGenerator atomic.Int64
	clientSecret       *secretmanager.Client
	clientSub          *pubsub.Client
	ctx                context.Context
}

func SetupSecrets(ctx context.Context, secrets []string) (*SecretStore, error) {

	// Debug mode: do not connect to Google Secret Manager
	if gin.Mode() == gin.DebugMode {
		return nil, nil
	}

	var s SecretStore = SecretStore{
		ctx:        ctx,
		secrets:    make(map[string]*atomic.Pointer[SecretValue]),
		watchers:   make(map[string][]SecretWatcher),
		watchersMu: sync.Mutex{},
	}

	var err error
	s.clientSecret, err = secretmanager.NewClient(s.ctx)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		value, err := s.retrieveSecret(s.ctx, secret)
		if err != nil {
			return nil, err
		}

		secretObj := &SecretValue{
			Value:       value,
			RetrievedAt: time.Now(),
		}
		s.secrets[secret] = &atomic.Pointer[SecretValue]{}
		s.secrets[secret].Store(secretObj)
	}

	err = messages.SubscribeUpdatesChannel(s.ctx, "API_SECRET_CHANGE", s.listener)

	if err != nil {
		panic(err)
	}

	return &s, nil
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
		return err
	}

	if strings.Count(update.SecretId, "/") != 3 {
		return fmt.Errorf("Secret ID %s is not in format `projects/{project}/secrets/{secret}`", update.SecretId)
	}

	secretFullnameParts := strings.Split(update.SecretId, "/")
	secretName := secretFullnameParts[3]

	if ptr, exists := s.secrets[secretName]; exists {
		value, err := s.retrieveSecret(ctx, secretName)
		if err != nil {
			panic(err)
		}
		secretObj := &SecretValue{
			Value:       value,
			RetrievedAt: time.Now(),
		}
		ptr.Store(secretObj)
		// Notify watchers
		s.watchersMu.Lock()
		for _, w := range s.watchers[secretName] {
			go w.Handler(*secretObj)
		}
		s.watchersMu.Unlock()
	}
	return nil
}

func (s *SecretStore) Get(name string) *SecretValue {
	osEnv := os.Getenv(name)
	if osEnv != "" {
		return &SecretValue{Value: osEnv, RetrievedAt: time.Now()}
	}
	ptr, exists := s.secrets[name]
	if !exists {
		return nil
	}
	secret := ptr.Load()
	if secret == nil {
		return nil
	}
	return secret
}

func (s *SecretStore) Watch(name string, handler func(SecretValue)) (cancel func()) {
	id := s.watcherIdGenerator.Add(1)
	w := SecretWatcher{
		ID:      &id,
		Handler: handler,
	}
	s.watchersMu.Lock()
	s.watchers[name] = append(s.watchers[name], w)
	s.watchersMu.Unlock()

	return func() {
		s.watchersMu.Lock()
		defer s.watchersMu.Unlock()
		watchers := s.watchers[name]
		for i, w := range watchers {
			if w.ID == &id {
				s.watchers[name] = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
	}
}

func (s *SecretStore) retrieveSecret(ctx context.Context, name string) (string, error) {
	projectID := google.GetNumericProjectID()
	result, err := s.clientSecret.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name),
	})
	if err != nil {
		return "", err
	}
	return string(result.Payload.Data), nil
}
