package main

import (
	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/google"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/server"
	"auto-myself-api/internal/storage"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var requiredSecrets = []string{
	"POSTGRES_DSN",
	"GOOGLE_OAUTH2_CLIENT_IDS",
	"APPLE_SIGN_IN_CLIENT_IDS",
	"JWT_SIGNING_SECRET",
}

type signedURLVerifier interface {
	VerifySignedURL(ctx context.Context) error
}

var _ signedURLVerifier = (*storage.GCS)(nil)

func verifyStorageSigning(ctx context.Context, mode string, verifier signedURLVerifier) error {
	if mode != gin.ReleaseMode {
		return nil
	}
	if err := verifier.VerifySignedURL(ctx); err != nil {
		return fmt.Errorf("verifying storage signing: %w", err)
	}
	return nil
}

func Setup(ctx context.Context, r *gin.Engine) (func(), error) {
	secretStore, err := secrets.SetupSecrets(ctx, requiredSecrets)
	if err != nil {
		return nil, fmt.Errorf("setting up secrets: %w", err)
	}

	googleClientIDs, err := secretList(secretStore, "GOOGLE_OAUTH2_CLIENT_IDS")
	if err != nil {
		_ = secretStore.Close()
		return nil, fmt.Errorf("reading GOOGLE_OAUTH2_CLIENT_IDS: %w", err)
	}
	appleClientIDs, err := secretList(secretStore, "APPLE_SIGN_IN_CLIENT_IDS")
	if err != nil {
		_ = secretStore.Close()
		return nil, fmt.Errorf("reading APPLE_SIGN_IN_CLIENT_IDS: %w", err)
	}
	identityVerifiers, err := auth.NewVerifiers(googleClientIDs, appleClientIDs)
	if err != nil {
		_ = secretStore.Close()
		return nil, fmt.Errorf("configuring identity verifiers: %w", err)
	}

	db, err := CreateSelfUpdatingDatabase(ctx, secretStore)
	if err != nil {
		_ = secretStore.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	bucketName := google.GetBucketName()
	bucketStorage := storage.NewGCS(bucketName)
	mode := gin.Mode()

	if err := verifyStorageSigning(ctx, mode, bucketStorage); err != nil {
		_ = bucketStorage.Close()
		_ = secretStore.Close()
		_ = db.Close()
		return nil, fmt.Errorf("verifying storage for bucket %s: %w", bucketName, err)
	}
	if mode == gin.ReleaseMode {
		slog.Info("storage signed-URL signing verified", "bucket", bucketName)
	}

	deps := server.Dependencies{
		Database:          db,
		Secrets:           secretStore,
		Storage:           bucketStorage,
		IdentityVerifiers: identityVerifiers,
	}
	server.RegisterRoutes(r, server.Build(deps), deps)

	shutdown := func() {
		if err := bucketStorage.Close(); err != nil {
			slog.Error("failed to close storage client", "error", err)
		}
		if err := secretStore.Close(); err != nil {
			slog.Error("failed to close secret store", "error", err)
		}
		if err := db.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}

	return shutdown, nil
}

func CreateSelfUpdatingDatabase(ctx context.Context, s *secrets.SecretStore) (*database.Database, error) {
	const dsnSecret = "POSTGRES_DSN"

	dsn, err := s.Get(dsnSecret)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dsnSecret, err)
	}

	db, err := database.ConnectPostgres(ctx, dsn.Value)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	wrapper, err := database.NewDatabase(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating database wrapper: %w", err)
	}

	s.Watch(dsnSecret, func(value secrets.SecretValue) {
		reconnectWithRetry(ctx, wrapper, value.Value)
	})

	return wrapper, nil
}

func secretList(store *secrets.SecretStore, name string) ([]string, error) {
	value, err := store.Get(name)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var list []string
	for _, part := range strings.Split(value.Value, ",") {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		list = append(list, item)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("%s must contain at least one client ID", name)
	}

	return list, nil
}

// Swaps in a fresh connection after the DSN rotates.
func reconnectWithRetry(ctx context.Context, wrapper *database.Database, dsn string) {
	const attempts = 5
	backoff := time.Second

	for attempt := 1; attempt <= attempts; attempt++ {
		newDB, err := database.ConnectPostgres(ctx, dsn)
		if err == nil {
			oldDB, replaceErr := wrapper.Replace(newDB)
			if replaceErr != nil {
				_ = newDB.Close()
				slog.Error("failed to swap in rotated database connection", "error", replaceErr)
				return
			}
			if oldDB != nil {
				if closeErr := oldDB.Close(); closeErr != nil {
					slog.Error("failed to close replaced database connection", "error", closeErr)
				}
			}
			return
		}

		slog.Error("failed to reconnect after DSN rotation", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff *= 2
	}
}
