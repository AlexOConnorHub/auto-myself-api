package controllers

import (
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

var allUsers = [][]string{
	{"019785fe-4eb4-766e-9c45-bec7780972a2", "User 1"},
	{"019785fe-4eb4-766e-9c45-c1f83e7c1f1f", "User 2"},
	{"019785fe-4eb4-766e-9c45-c497f2d9fe9e", "User 3"},
	{"019785fe-4eb4-766e-9c45-c8578456b4df", "User 4"},
	{"019785fe-4eb4-766e-9c45-cec136a9ad6f", "User 5"},
	{"019785fe-4eb4-766e-9c45-f592a1187d0c", "User 6"},
	{"019785fe-4eb4-766e-9c45-f9cd4ee5c0b3", "User 7"},
	{"019785fe-4eb4-766e-9c45-fc6ed4a7407b", "User 8"},
}

func TestVerifyAllUsersExist(t *testing.T) {
	r := setupTest(t)

	// for uuid, expectedUsername := range allUsers {
	for _, user := range allUsers {
		uuid := user[0]
		expectedUsername := user[1]

		w := helpers.PerformRequest(r, "GET", "/user", map[string]string{"auth_uuid": uuid, "content-type": "application/json"}, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		var firstBody = w.Body.String()

		var user models.UserBase
		if err := json.Unmarshal(w.Body.Bytes(), &user); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if user.Username != expectedUsername {
			t.Error("Expected username to be", expectedUsername, ", got", user.Username)
		}

		w = helpers.PerformRequest(r, "GET", "/user/"+uuid, map[string]string{"auth_uuid": uuid, "content-type": "application/json"}, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		if w.Body.String() != firstBody {
			t.Error("/user/" + uuid + " endpoint: Expected response body to be the same as /user endpoint")
		}
	}
}

func TestModifyUser(t *testing.T) {
	r := setupTest(t)

	var testUser = allUsers[0]
	uuid := testUser[0]
	initialUsername := testUser[1]

	var testUserModel models.UserBase
	var userResponse models.UserBase

	testUserModel.Username = initialUsername

	w := helpers.PerformRequest(r, "GET", "/user", map[string]string{"auth_uuid": uuid, "content-type": "application/json"}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &userResponse); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if userResponse.Username != initialUsername {
		t.Error("Expected username to be", initialUsername, ", got", userResponse.Username)
	}

	testUserModel.Username = "MODIFIED " + initialUsername

	bodyBytes, err := json.Marshal(testUserModel)
	if err != nil {
		t.Fatalf("Failed to marshal testUserModel: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w = helpers.PerformRequest(r, "PATCH", "/user", map[string]string{"auth_uuid": uuid, "content-type": "application/json"}, bodyReader)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &userResponse); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if userResponse.Username != testUserModel.Username {
		t.Error("Expected username to be", testUserModel.Username, ", got", userResponse.Username)
	}

	w = helpers.PerformRequest(r, "GET", "/user/"+uuid, map[string]string{"auth_uuid": uuid, "content-type": "application/json"}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &userResponse); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if userResponse.Username != testUserModel.Username {
		t.Error("Expected username to be", testUserModel.Username, ", got", userResponse.Username)
	}
}

func TestReadUserPermissions(t *testing.T) {
	r := setupTest(t)

	/*
		Each row is a different user. Each column is the ability of the user to view the profile of the other uesrs.
		Rules:
			Vehicle creator can view all who have any access to vehicle
			Users with access to a vehicle can view users with write access (including vehicle creator)
	*/
	readAccessMatrix := [8][8]bool{
		{true, true, true, false, false, false, true, true},
		{true, true, false, false, false, false, true, true},
		{true, true, true, false, false, false, true, true},
		{false, false, false, true, false, false, false, false},
		{false, false, false, false, true, false, false, false},
		{false, false, false, false, false, true, true, true},
		{true, true, false, false, false, true, true, true},
		{true, true, false, false, false, true, true, true},
	}

	var successMatrix [8][8]string

	for authUser, acces := range readAccessMatrix {
		for readUser, canRead := range acces {
			auth_uuid := allUsers[authUser][0]
			read_uuid := allUsers[readUser][0]
			read_username := allUsers[readUser][1]

			w := helpers.PerformRequest(r, "GET", "/user/"+read_uuid, map[string]string{"auth_uuid": auth_uuid, "content-type": "application/json"}, nil)
			if w.Code == http.StatusOK {
				if !canRead {
					successMatrix[authUser][readUser] = fmt.Sprintf("Expected %d but got %d", http.StatusNotFound, w.Code)
					continue
				}
				var userResponse models.UserBase
				if err := json.Unmarshal(w.Body.Bytes(), &userResponse); err != nil {
					successMatrix[authUser][readUser] = fmt.Sprintf("Failed to unmarshal response: %v", err)
					continue
				}

				if userResponse.Username != read_username {
					successMatrix[authUser][readUser] = fmt.Sprintf("Expected to have username %s, got %s", read_username, userResponse.Username)
					continue
				}
			} else if w.Code == http.StatusNotFound {
				if canRead {
					successMatrix[authUser][readUser] = fmt.Sprintf("Expected %d but got %d", http.StatusOK, w.Code)
					continue
				}
			} else {
				successMatrix[authUser][readUser] = fmt.Sprintf("Unexpected status code %d for %s reading %s", w.Code, auth_uuid, read_uuid)
				continue
			}
		}
	}

	for authUser, acces := range successMatrix {
		for readUser, errorMessage := range acces {
			if errorMessage != "" {
				authUserName := allUsers[authUser][1]
				readUserName := allUsers[readUser][1]
				t.Errorf("Auth User %s reading User %s: %s", authUserName, readUserName, errorMessage)
			}
		}
	}
}
