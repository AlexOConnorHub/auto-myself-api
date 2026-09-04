package controllers

import (
	"auto-myself-api/helpers"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestReport(t *testing.T) {
	r, _ := setupTest(t)

	userUuid := AllUsers[0][0]

	body := map[string]interface{}{
		"reported_table": "users",
		"reported_id":    AllUsers[1][0],
		"reason":         "Test reason",
	}

	bodyReader := bytes.NewReader([]byte{})
	if b, err := json.Marshal(body); err == nil {
		bodyReader = bytes.NewReader(b)
	}

	w := helpers.TestRequestAsUser(r, "POST", "/v1/report", userUuid, bodyReader)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d for report creation, got %d\n%s", http.StatusCreated, w.Code, w.Body.String())
	}
}
