package google

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	_ "github.com/joho/godotenv/autoload"
)

var (
	projectMu         sync.Mutex
	gProjectID        string
	gProjectNumericID string
)

func metadataValue(path string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/"+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server returned %s for %s", resp.Status, path)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func GetProjectID() string {
	projectMu.Lock()
	defer projectMu.Unlock()

	if gProjectID != "" {
		return gProjectID
	}

	if envProjectID := os.Getenv("GOOGLE_CLOUD_PROJECT"); envProjectID != "" {
		gProjectID = envProjectID
		return gProjectID
	}

	value, err := metadataValue("project/project-id")
	if err != nil {
		panic("failed to determine Google Cloud project ID: " + err.Error())
	}
	gProjectID = value
	return gProjectID
}

func GetNumericProjectID() string {
	projectMu.Lock()
	defer projectMu.Unlock()

	if gProjectNumericID != "" {
		return gProjectNumericID
	}

	if envID := os.Getenv("GOOGLE_CLOUD_PROJECT_NUMERIC"); envID != "" {
		gProjectNumericID = envID
		return gProjectNumericID
	}

	value, err := metadataValue("project/numeric-project-id")
	if err != nil {
		panic("failed to determine Google Cloud numeric project ID: " + err.Error())
	}
	gProjectNumericID = value
	return gProjectNumericID
}

func GetBucketName() string {
	if bucket := os.Getenv("STORAGE_BUCKET"); bucket != "" {
		return bucket
	}
	return "staging." + GetProjectID() + ".appspot.com"
}
