package controllers

import (
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofrs/uuid/v5"
)

var AllShares = [][]interface{}{
	{"01988201-ec4e-7bae-88eb-17fd5bf60ca8", "019785fe-4eb4-766e-9c45-cec136a9ad6f", "019785fe-4eb4-766e-9c45-ddfb4b2e7210", true, "019785fe-4eb4-766e-9c45-f592a1187d0c"},
	{"019d5352-0a33-78e5-9148-59bac3670a1a", "019785fe-4eb4-766e-9c45-c8578456b4df", "019785fe-4eb4-766e-9c45-ddfb4b2e7210", false, "019785fe-4eb4-766e-9c45-f592a1187d0c"},
}

// "READ_ONLY" really means can read and delete.
// "WRITE" means can accept as well
var ShareAccessMatrix = [8][2]int{
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, WRITE},
	{WRITE, NO_ACCESS},
	{READ_ONLY, READ_ONLY},
	{NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS},
}

func TestShareVerifyAllExist(t *testing.T) {
	r, _ := setupTest(t)

	for _, shareRow := range AllShares {
		uuid := shareRow[0].(string)
		userUUID := shareRow[4].(string)

		w := helpers.TestRequestAsUser(r, "GET", "/v1/share/"+uuid, userUUID, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
		}
	}
}

func TestShareCreate(t *testing.T) {
	r, _ := setupTest(t)

	userUUID, err := uuid.FromString(AllUsers[7][0])
	if err != nil {
		t.Fatalf("Failed to parse user UUID: %v", err)
	}

	vehicleUUID, err := uuid.FromString(AllVehicles[0][0].(string))
	if err != nil {
		t.Fatalf("Failed to parse vehicle UUID: %v", err)
	}

	newShare := models.VehicleUserAccessBase{
		UserID:      userUUID,
		VehicleID:   vehicleUUID,
		WriteAccess: true,
	}

	jsonBody, err := json.Marshal(newShare)
	if err != nil {
		t.Fatalf("Failed to marshal new share to JSON: %v", err)
	}

	w := helpers.TestRequestAsUser(r, "POST", "/v1/share", "019785fe-4eb4-766e-9c45-f592a1187d0c", bytes.NewBuffer(jsonBody))
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestShareAccept(t *testing.T) {
	r, a := setupTest(t)

	var successMatrix [8][2]string

	for authUserIndex, access := range ShareAccessMatrix {
		for shareIndex, permission := range access {
			authUUID := AllUsers[authUserIndex][0]
			shareUUID := AllShares[shareIndex][0].(string)

			a.Gorm.SavePoint("TestShareAccept")
			w := helpers.TestRequestAsUser(r, "PATCH", "/v1/share/"+shareUUID, authUUID, nil)
			a.Gorm.RollbackTo("TestShareAccept")

			if w.Code == http.StatusNoContent {
				if permission != WRITE {
					successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusNotFound, w.Code)
					continue
				}
			} else if w.Code == http.StatusNotFound {
				if permission == WRITE {
					successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusNoContent, w.Code)
					continue
				}
			} else if w.Code == http.StatusForbidden {
				if permission != READ_ONLY {
					successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusForbidden, w.Code)
					continue
				}
			} else {
				successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Unexpected status code %d for %s writing %s", w.Code, authUUID, shareUUID)
				continue
			}
		}
	}

	for authUser, access := range successMatrix {
		for shareIndex, errorMessage := range access {
			if errorMessage != "" {
				authUserName := fmt.Sprintf("User %d", authUser+1)
				shareUUID := AllShares[shareIndex][0].(string)
				t.Errorf("Auth User %s writing Share %s: %s", authUserName, shareUUID, errorMessage)
			}
		}
	}
}

func TestShareReadAccess(t *testing.T) {
	r, _ := setupTest(t)

	var successMatrix [8][2]string

	for authUserIndex, accesses := range ShareAccessMatrix {
		for shareIndex, permission := range accesses {
			authUUID := AllUsers[authUserIndex][0]
			shareUUID := AllShares[shareIndex][0].(string)

			w := helpers.TestRequestAsUser(r, "GET", "/v1/share/"+shareUUID, authUUID, nil)
			if w.Code == http.StatusOK {
				if permission == NO_ACCESS {
					successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusNotFound, w.Code)
					continue
				}
			} else if w.Code == http.StatusNotFound {
				if permission > NO_ACCESS {
					successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusOK, w.Code)
					continue
				}
			} else {
				successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Unexpected status code %d for %s reading %s", w.Code, authUUID, shareUUID)
				continue
			}
		}
	}

	for authUser, access := range successMatrix {
		for shareIndex, errorMessage := range access {
			if errorMessage != "" {
				authUserName := fmt.Sprintf("User %d", authUser+1)
				shareUUID := AllShares[shareIndex][0].(string)
				t.Errorf("Auth User %s reading Share %s: %s", authUserName, shareUUID, errorMessage)
			}
		}
	}
}

func TestShareDelete(t *testing.T) {
	r, a := setupTest(t)

	var successMatrix [8][2]string

	for authUserIndex, access := range ShareAccessMatrix {
		for shareIndex, permission := range access {
			authUUID := AllUsers[authUserIndex][0]
			shareUUID := AllShares[shareIndex][0].(string)

			a.Gorm.SavePoint("TestShareDelete")
			w := helpers.TestRequestAsUser(r, "DELETE", "/v1/share/"+shareUUID, authUUID, nil)
			a.Gorm.RollbackTo("TestShareDelete")

			if w.Code == http.StatusNoContent {
				if permission < READ_ONLY {
					successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusNotFound, w.Code)
					continue
				}
			} else if w.Code == http.StatusNotFound {
				if permission >= READ_ONLY {
					successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusNoContent, w.Code)
					continue
				}
			} else {
				successMatrix[authUserIndex][shareIndex] = fmt.Sprintf("Unexpected status code %d for %s writing %s", w.Code, authUUID, shareUUID)
				continue
			}
		}
	}

	for authUser, access := range successMatrix {
		for shareIndex, errorMessage := range access {
			if errorMessage != "" {
				authUserName := fmt.Sprintf("User %d", authUser+1)
				shareUUID := AllShares[shareIndex][0].(string)
				t.Errorf("Auth User %s deleting Share %s: %s", authUserName, shareUUID, errorMessage)
			}
		}
	}
}

func TestShareList(t *testing.T) {
	r, _ := setupTest(t)

	for userIndex, accessRow := range ShareAccessMatrix {
		authUUID, err := uuid.FromString(AllUsers[userIndex][0])
		if err != nil {
			t.Fatalf("Failed to parse user UUID: %v", err)
		}

		expectedShareCount := 0
		for _, access := range accessRow {
			if access > NO_ACCESS {
				expectedShareCount++
			}
		}

		w := helpers.TestRequestAsUser(r, "GET", "/v1/share", authUUID.String(), nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			continue
		}

		var shares []string
		if err := json.Unmarshal(w.Body.Bytes(), &shares); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			continue
		}

		if len(shares) != expectedShareCount {
			t.Errorf("Expected %d shares, got %d", expectedShareCount, len(shares))
		}
	}
}
