package helpers

import (
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	_ "github.com/joho/godotenv/autoload"
)

var secrets = map[string]string{}
var gProjectID string
var gProjectNumericID string

func getProjectID() string {
	if gProjectID != "" {
		return gProjectID
	}

	envProjectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if envProjectID != "" {
		gProjectID = envProjectID
		return gProjectID
	}

	data, err := http.Get("http://metadata.google.internal/computeMetadata/v1/project/project-id")
	if err != nil {
		panic("Failed to determine Google Cloud project ID from metadata server")
	}

	defer data.Body.Close()
	body, err := io.ReadAll(data.Body)
	if err != nil {
		panic("Failed to read response from metadata server: " + err.Error())
	}

	gProjectID = string(body)
	return gProjectID
}

func getNumericProjectID() string {
	if gProjectNumericID != "" {
		return gProjectNumericID
	}

	envProjectNumericID := os.Getenv("GOOGLE_CLOUD_PROJECT_NUMERIC")
	if envProjectNumericID != "" {
		gProjectNumericID = envProjectNumericID
		return gProjectNumericID
	}

	data, err := http.Get("http://metadata.google.internal/computeMetadata/v1/project/numeric-project-id")
	if err != nil {
		panic("Failed to determine Google Cloud project numeric ID from metadata server")
	}

	defer data.Body.Close()
	body, err := io.ReadAll(data.Body)
	if err != nil {
		panic("Failed to read response from metadata server: " + err.Error())
	}

	gProjectNumericID = string(body)
	return gProjectNumericID
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

	projectID := getNumericProjectID()

	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, name),
	})
	client.Close()
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

func getBucketName() string {
	projectID := getProjectID()
	return "staging." + projectID + ".appspot.com"
}

func GetFileUrl(filePath string) (string, error) {
	bucketName := getBucketName()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	expirationTime := time.Now().Add(15 * time.Minute)
	if url, err := bucket.SignedURL(filePath, &storage.SignedURLOptions{
		Expires: expirationTime,
		Method:  "GET",
	}); err != nil {
		return "", err
	} else {
		return url, nil
	}
}

func DeleteFile(filePath string) error {
	bucketName := getBucketName()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Bucket(bucketName).Object(filePath).Delete(ctx)
}

func CreateFile(filePath string, data *[]byte) error {
	bucketName := getBucketName()
	println("Creating file at path: " + filePath + " in bucket: " + bucketName)

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	wc := client.Bucket(bucketName).Object(filePath).NewWriter(ctx)
	if _, err := wc.Write(*data); err != nil {
		return err
	}
	return wc.Close()
}
