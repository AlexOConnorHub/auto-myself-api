package helpers

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	_ "github.com/joho/godotenv/autoload"
)

var secrets = map[string]string{}
var ctx = context.Background()

func ConnectGoogleClient() *storage.Client {
	client, err := storage.NewClient(ctx)
	if err != nil {
		panic("failed to create Google Cloud Storage client: " + err.Error())
	}
	return client
}

func GetSecret(name string) string {
	secret, exists := secrets[name]
	if exists {
		return secret
	}

	secret = os.Getenv(name)
	if secret != "" {
		secrets[name] = secret
		return secret
	}

	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		panic("failed to create Google Cloud Secret Manager client: " + err.Error())
	}
	defer client.Close()

	projectID := os.Getenv("GOOGLE_SECRET_MANAGER_PROJECT_ID")
	if projectID == "" {
		panic("GOOGLE_SECRET_MANAGER_PROJECT_ID environment variable is not set")
	}

	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name),
	})
	if err != nil {
		panic("failed to locate secret " + name + ": " + err.Error())
	}

	crc32c := crc32.MakeTable(crc32.Castagnoli)
	checksum := int64(crc32.Checksum(result.Payload.Data, crc32c))
	if checksum != *result.Payload.DataCrc32C {
		panic("Data corruption detected.")
	}

	secrets[name] = string(result.Payload.Data)
	return secrets[name]
}
