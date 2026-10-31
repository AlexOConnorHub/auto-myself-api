package main

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/google"
	"auto-myself-api/internal/secrets"
	"auto-myself-api/internal/server"
	"auto-myself-api/internal/storage"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

var requiredSecrets = []string{
	"DOMAIN",
	"POSTGRES_DSN",
	"GOOGLE_OAUTH2_CLIENT_ID",
	"GOOGLE_OAUTH2_CLIENT_SECRET",
	"JWT_SIGNING_SECRET",
}

func Setup(ctx context.Context, r *gin.Engine) (func(), error) {
	secretStore, err := secrets.SetupSecrets(ctx, requiredSecrets)
	if err != nil {
		return nil, fmt.Errorf("setting up secrets: %w", err)
	}

	db, err := CreateSelfUpdatingDatabase(ctx, secretStore)
	if err != nil {
		_ = secretStore.Close()
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	bucketStorage := storage.NewGCS(google.GetBucketName())

	deps := server.Dependencies{
		Database: db,
		Secrets:  secretStore,
		Storage:  bucketStorage,
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
