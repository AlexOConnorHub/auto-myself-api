package helpers

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/pubsub/v2"
	// pubsub "cloud.google.com/go/pubsub/v2"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
)

type SecretValue struct {
	Value       string
	Version     string
	RetrievedAt time.Time
}

type SecretWatcher struct {
	id      *int64
	handler func(SecretValue)
}

type SecretStore struct {
	secrets            map[string]*atomic.Pointer[SecretValue]
	watchers           map[string][]SecretWatcher
	watchersMu         sync.Mutex
	watcherIdGenerator atomic.Int64
}

// var secrets = map[string]atomic.Pointer[SecretValue]{
// 	"DOMAIN"
// 	"KEYRING"
// 	"POSTGRES_DSN"
// 	"GOOGLE_OAUTH2_CLIENT_ID"
// 	"GOOGLE_OAUTH2_CLIENT_SECRET"
// }

var clientSecret *secretmanager.Client
var clientSub *pubsub.Client
var ctx context.Context = context.Background()

func retrieveSecret(name string) (string, error) {
	projectID := getNumericProjectID()
	result, err := clientSecret.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name),
	})
	if err != nil {
		return "", err
	}
	return string(result.Payload.Data), nil
}

func SetupSecrets(secrets []string) (*SecretStore, error) {

	// Debug mode: do not connect to Google Secret Manager
	if gin.Mode() == gin.DebugMode {
		return nil, nil
	}

	var secretStoreHouse SecretStore = SecretStore{
		secrets:    make(map[string]*atomic.Pointer[SecretValue]),
		watchers:   make(map[string][]SecretWatcher),
		watchersMu: sync.Mutex{},
	}

	var err error
	clientSecret, err = secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	for _, secret := range secrets {
		// req := &secretmanager.AccessSecretVersionRequest{
		// 	Name: secret,
		// }

		value, err := retrieveSecret(secret)
		if err != nil {
			return nil, err
		}
		secretObj := &SecretValue{
			Value:       value,
			RetrievedAt: time.Now(),
		}
		secretStoreHouse.secrets[secret] = &atomic.Pointer[SecretValue]{}
		secretStoreHouse.secrets[secret].Store(secretObj)
	}

	projectID := getNumericProjectID()
	clientSub, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	defer clientSub.Close()

	sub := clientSub.Subscriber("API_SECRET_UPDATES")

	err = sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		// If message is updated secret and secret is in the store, update it and notify watchers
		if msg.Attributes["secret_name"] != "" {
			secretName := msg.Attributes["secret_name"]
			if ptr, exists := secretStoreHouse.secrets[secretName]; exists {
				value, err := retrieveSecret(secretName)
				if err != nil {
					panic(err)
				}
				secretObj := &SecretValue{
					Value:       value,
					RetrievedAt: time.Now(),
				}
				ptr.Store(secretObj)
				// Notify watchers
				secretStoreHouse.watchersMu.Lock()
				for _, w := range secretStoreHouse.watchers[secretName] {
					go w.handler(*secretObj)
				}
				secretStoreHouse.watchersMu.Unlock()
			}
		}

		msg.Ack()
	})
	if err != nil {
		return nil, err
	}
	return &secretStoreHouse, nil
}

func (s *SecretStore) Get(name string) (SecretValue, error) {
	osEnv := os.Getenv(name)
	if osEnv != "" {
		return SecretValue{Value: osEnv, RetrievedAt: time.Now()}, nil
	}
	ptr, exists := s.secrets[name]
	if !exists {
		return SecretValue{}, fmt.Errorf("secret not found: %s", name)
	}
	secret := ptr.Load()
	if secret == nil {
		return SecretValue{}, fmt.Errorf("secret not loaded: %s", name)
	}
	return *secret, nil
}

func (s *SecretStore) Watch(name string, handler func(SecretValue)) (cancel func()) {
	id := s.watcherIdGenerator.Add(1)
	w := SecretWatcher{
		id:      &id,
		handler: handler,
	}
	s.watchersMu.Lock()
	s.watchers[name] = append(s.watchers[name], w)
	s.watchersMu.Unlock()

	return func() {
		s.watchersMu.Lock()
		defer s.watchersMu.Unlock()
		watchers := s.watchers[name]
		for i, w := range watchers {
			if w.id == &id {
				s.watchers[name] = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
	}
}
