package controllers

import (
	"auto-myself-api/helpers"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestFiles(t *testing.T) {
	r, _ := setupTest(t)

	userUuid := AllUsers[0][0]
	maintenanceUuid := AllMaintenances[0][0].(string)

	bodyReader := bytes.NewReader([]byte(`{"isFile": true}`))

	w := helpers.TestRequestAsUser(r, "POST", "/v1/file/maintenance/"+maintenanceUuid, userUuid, bodyReader)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d for file upload, got %d\n%s", http.StatusCreated, w.Code, w.Body.String())
	}

	w = helpers.TestRequestAsUser(r, "GET", "/v1/file/maintenance/"+maintenanceUuid, userUuid, nil)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d for files list, got %d\n%s", http.StatusOK, w.Code, w.Body.String())
	}

	data := []struct {
		ID  string `json:"id"`
		URL string `json:"signed_url"`
	}{}
	_ = json.Unmarshal(w.Body.Bytes(), &data)

	googleResp, err := http.Get(data[0].URL)
	if err != nil {
		t.Errorf("Failed to GET signed URL: %v", err)
	}
	defer googleResp.Body.Close()
	if googleResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d for signed URL, got %d", http.StatusOK, googleResp.StatusCode)
	}

	w = helpers.TestRequestAsUser(r, "DELETE", "/v1/file/"+data[0].ID, userUuid, nil)
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d for file delete, got %d\n%s", http.StatusNoContent, w.Code, w.Body.String())
	}
}
