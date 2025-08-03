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

var AllUsers = [][]string{
	{"019785fe-4eb4-766e-9c45-bec7780972a2", "User 1"},
	{"019785fe-4eb4-766e-9c45-c1f83e7c1f1f", "User 2"},
	{"019785fe-4eb4-766e-9c45-c497f2d9fe9e", "User 3"},
	{"019785fe-4eb4-766e-9c45-c8578456b4df", "User 4"},
	{"019785fe-4eb4-766e-9c45-cec136a9ad6f", "User 5"},
	{"019785fe-4eb4-766e-9c45-f592a1187d0c", "User 6"},
	{"019785fe-4eb4-766e-9c45-f9cd4ee5c0b3", "User 7"},
	{"019785fe-4eb4-766e-9c45-fc6ed4a7407b", "User 8"},
}

var UserAccessMatrix = [8][8]int{
	{WRITE, READ_ONLY, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY},
	{READ_ONLY, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY},
	{READ_ONLY, READ_ONLY, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, READ_ONLY, READ_ONLY},
	{READ_ONLY, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, WRITE, READ_ONLY},
	{READ_ONLY, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS, READ_ONLY, READ_ONLY, WRITE},
}

func validateUserResponse(marshaledResponse models.UserBase, expected models.UserBase) string {
	if marshaledResponse.Username != expected.Username {
		return fmt.Sprintf("Expected username to be %s, got %s", expected.Username, marshaledResponse.Username)
	}
	return ""
}

func loadUser(index int) models.UserBase {
	user := AllUsers[index]
	return models.UserBase{
		Username: user[1],
	}
}

func TestUsersVerifyAllExist(t *testing.T) {
	r := setupTest(t)

	for index, userRow := range AllUsers {
		uuid := userRow[0]
		user := loadUser(index)

		w := helpers.PerformRequest(r, "GET", "/user", map[string]string{"auth_uuid": uuid, "content-type": "application/json"}, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}

		var firstBody = w.Body.String()

		var response models.UserBase
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if errMsg := validateUserResponse(response, user); errMsg != "" {
			t.Error("User response validation failed:", errMsg)
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

func TestUserPatch(t *testing.T) {
	r := setupTest(t)

	userRow := AllUsers[0]
	uuid := userRow[0]
	user := loadUser(0)
	modified := models.UserBase{
		Username: "MODIFIED " + user.Username,
	}
	user.Username = modified.Username

	bodyBytes, err := json.Marshal(modified)
	if err != nil {
		t.Fatalf("Failed to marshal testUserModel: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w := helpers.PerformRequest(r, "PATCH", "/user", map[string]string{"auth_uuid": uuid, "content-type": "application/json"}, bodyReader)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response models.UserBase
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if errMsg := validateUserResponse(response, user); errMsg != "" {
		t.Error("User response validation failed:", errMsg)
	}
}

func TestUserReadPermissions(t *testing.T) {
	r := setupTest(t)

	var successMatrix [8][8]string

	for authUser, access := range UserAccessMatrix {
		for readUser, permission := range access {
			auth_uuid := AllUsers[authUser][0]
			read_uuid := AllUsers[readUser][0]
			expected := loadUser(readUser)

			w := helpers.PerformRequest(r, "GET", "/user/"+read_uuid, map[string]string{"auth_uuid": auth_uuid, "content-type": "application/json"}, nil)
			if w.Code == http.StatusOK {
				if permission == NO_ACCESS {
					successMatrix[authUser][readUser] = fmt.Sprintf("Expected %d but got %d", http.StatusNotFound, w.Code)
					continue
				}
				var response models.UserBase
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					successMatrix[authUser][readUser] = fmt.Sprintf("Failed to unmarshal response: %v", err)
					continue
				}

				if errMsg := validateUserResponse(response, expected); errMsg != "" {
					successMatrix[authUser][readUser] = fmt.Sprintf("User response validation failed: %s", errMsg)
					continue
				}
			} else if w.Code == http.StatusNotFound {
				if permission > NO_ACCESS {
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
				authUserName := AllUsers[authUser][1]
				readUserName := AllUsers[readUser][1]
				t.Errorf("Auth User %s reading User %s: %s", authUserName, readUserName, errorMessage)
			}
		}
	}
}
