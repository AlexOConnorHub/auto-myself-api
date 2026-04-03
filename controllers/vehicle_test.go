package controllers

import (
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"gorm.io/gorm"
)

var AllVehicles = [][]interface{}{
	{"019785fe-4eb4-766e-9c45-d0b2bb289b82", "Vehicle 1", 2015, "MAZDA", 473, "MX-5", 2072, "VIN CAR 1", "LPN CAR 1", "019785fe-4eb4-766e-9c45-bec7780972a2"},
	{"019785fe-4eb4-766e-9c45-d77f41aa8317", "Vehicle 2", 2021, "TOYOTA", 448, "Corolla", 2208, "VIN CAR 2", "LPN CAR 2", "019785fe-4eb4-766e-9c45-bec7780972a2"},
	{"019785fe-4eb4-766e-9c45-d9cc7ea628c1", "Vehicle 3", 2019, "FORD", 460, "Edge", 1797, "VIN CAR 3", "LPN CAR 3", "019785fe-4eb4-766e-9c45-c8578456b4df"},
	{"019785fe-4eb4-766e-9c45-ddfb4b2e7210", "Vehicle 4", 2018, "NISSAN", 478, "GT-R", 1890, "VIN CAR 4", "LPN CAR 4", "019785fe-4eb4-766e-9c45-f592a1187d0c"},
	{"019785fe-4eb4-766e-9c45-e1af5010246b", "Vehicle 5", 2025, "ASTON MARTIN", 440, "Vanquish", 1701, "VIN CAR 5", "LPN CAR 5", "019785fe-4eb4-766e-9c45-fc6ed4a7407b"},
}

var VehicleAccessMatrix = [8][5]int{
	{WRITE, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS},
	{READ_BY_INVITE, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, WRITE, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, READ_ONLY, WRITE},
}

func validateVehicleResponse(marshaledResponse models.VehicleBase, expected models.VehicleBase) string {
	if marshaledResponse.Nickname != expected.Nickname {
		return fmt.Sprintf("Expected nickname to be %s, got %s", expected.Nickname, marshaledResponse.Nickname)
	}
	if marshaledResponse.Year != expected.Year {
		return fmt.Sprintf("Expected year to be %d, got %d", expected.Year, marshaledResponse.Year)
	}
	if marshaledResponse.Make != expected.Make {
		return fmt.Sprintf("Expected make to be %s, got %s", expected.Make, marshaledResponse.Make)
	}
	if marshaledResponse.MakeID != expected.MakeID {
		return fmt.Sprintf("Expected makeID to be %d, got %d", expected.MakeID, marshaledResponse.MakeID)
	}
	if marshaledResponse.Model != expected.Model {
		return fmt.Sprintf("Expected model to be %s, got %s", expected.Model, marshaledResponse.Model)
	}
	if marshaledResponse.ModelID != expected.ModelID {
		return fmt.Sprintf("Expected modelID to be %d, got %d", expected.ModelID, marshaledResponse.ModelID)
	}
	if marshaledResponse.Vin != expected.Vin {
		return fmt.Sprintf("Expected vin to be %s, got %s", expected.Vin, marshaledResponse.Vin)
	}
	if marshaledResponse.Lpn != expected.Lpn {
		return fmt.Sprintf("Expected lpn to be %s, got %s", expected.Lpn, marshaledResponse.Lpn)
	}
	return ""
}

func loadVehicle(index int) models.VehicleBase {
	vehicle := AllVehicles[index]
	return models.VehicleBase{
		Nickname: vehicle[1].(string),
		Year:     vehicle[2].(int),
		Make:     vehicle[3].(string),
		MakeID:   vehicle[4].(int),
		Model:    vehicle[5].(string),
		ModelID:  vehicle[6].(int),
		Vin:      vehicle[7].(string),
		Lpn:      vehicle[8].(string),
	}
}

func TestVehiclesVerifyAllExist(t *testing.T) {
	r, _ := setupTest(t)

	for index, vehicle := range AllVehicles {
		expected := loadVehicle(index)
		w := helpers.TestRequestAsUser(r, "GET", "/v1/vehicle/"+vehicle[0].(string), vehicle[9].(string), nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			continue
		}
		var vehicle models.VehicleBase
		if err := json.Unmarshal(w.Body.Bytes(), &vehicle); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			continue
		}
		if errMsg := validateVehicleResponse(vehicle, expected); errMsg != "" {
			t.Error(errMsg)
			continue
		}
	}
}

func TestVehicleList(t *testing.T) {
	r, _ := setupTest(t)

	for index, userAccess := range VehicleAccessMatrix {
		authUUID := AllUsers[index][0]
		expectedVehicleCount := 0
		for _, access := range userAccess {
			if access > READ_BY_INVITE {
				expectedVehicleCount++
			}
		}
		w := helpers.TestRequestAsUser(r, "GET", "/v1/vehicle", authUUID, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d for user %s", http.StatusOK, w.Code, AllUsers[index][1])
			continue
		}

		var ids []string
		if err := json.Unmarshal(w.Body.Bytes(), &ids); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
		}

		if len(ids) != expectedVehicleCount {
			t.Errorf("Expected %d vehicles, got %d for user %s", expectedVehicleCount, len(ids), AllUsers[index][1])
		}
	}
}

func TestVehiclePost(t *testing.T) {
	r, _ := setupTest(t)

	newVehicle := models.VehicleBase{
		Nickname: "New Vehicle",
		Year:     2022,
		Make:     "TEST-Make",
		Model:    "TEST-Model",
		Vin:      "VIN-TEST-1234567890",
		Lpn:      "LPN-TEST-1234",
	}

	bodyBytes, err := json.Marshal(newVehicle)
	if err != nil {
		t.Fatalf("Failed to marshal new vehicle: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w := helpers.TestRequestAsUser(r, "POST", "/v1/vehicle", AllUsers[0][0], bodyReader)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header to be set, but it was empty")
	}

	w = helpers.TestRequestAsUser(r, "GET", location, AllUsers[0][0], nil)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var vehicleResponse models.VehicleBase
	if err := json.Unmarshal(w.Body.Bytes(), &vehicleResponse); err != nil {
		t.Errorf("Failed to unmarshal response: %v \n%s", err, w.Body.String())
	}

	if errMsg := validateVehicleResponse(vehicleResponse, newVehicle); errMsg != "" {
		t.Error(errMsg)
	}
}

func TestVehiclePostWithUUID(t *testing.T) {
	r, _ := setupTest(t)

	uuid := "019d437c-5f48-753b-8240-5263972f15f0"

	newVehicle := models.VehicleBase{
		Nickname: "New Vehicle",
		Year:     2022,
		Make:     "TEST-Make",
		Model:    "TEST-Model",
		Vin:      "VIN-TEST-1234567890",
		Lpn:      "LPN-TEST-1234",
	}

	bodyBytes, err := json.Marshal(newVehicle)
	if err != nil {
		t.Fatalf("Failed to marshal new vehicle: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w := helpers.TestRequestAsUser(r, "POST", "/v1/vehicle/"+uuid, AllUsers[0][0], bodyReader)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Error("Expected Location header to be set, but it was empty")
	}

	pathParts := strings.Split(location, "/")
	if pathParts[len(pathParts)-1] != uuid {
		t.Errorf("Expected vehicle ID to be %s, got %s", uuid, pathParts[len(pathParts)-1])
	}

	w = helpers.TestRequestAsUser(r, "GET", location, AllUsers[0][0], nil)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var vehicleResponse models.VehicleBase
	if err := json.Unmarshal(w.Body.Bytes(), &vehicleResponse); err != nil {
		t.Errorf("Failed to unmarshal response: %v \n%s", err, w.Body.String())
	}

	if errMsg := validateVehicleResponse(vehicleResponse, newVehicle); errMsg != "" {
		t.Error(errMsg)
	}
}

func TestVehiclePatch(t *testing.T) {
	r, _ := setupTest(t)

	var testVehicle = AllVehicles[0]
	authUUID := testVehicle[9].(string)
	expected := loadVehicle(0)
	expected.Nickname = "MODIFIED " + expected.Nickname
	modified := models.VehicleBase{
		Nickname: expected.Nickname,
	}

	bodyBytes, err := json.Marshal(modified)
	if err != nil {
		t.Fatalf("Failed to marshal modified vehicle: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w := helpers.TestRequestAsUser(r, "PATCH", "/v1/vehicle/"+testVehicle[0].(string), authUUID, bodyReader)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var vehicleResponse models.VehicleBase
	if err := json.Unmarshal(w.Body.Bytes(), &vehicleResponse); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if errMsg := validateVehicleResponse(vehicleResponse, expected); errMsg != "" {
		t.Error(errMsg)
	}
}

func TestVehicleDelete(t *testing.T) {
	r, a := setupTest(t)

	var testVehicle = AllVehicles[0]
	authUUID := testVehicle[9].(string)

	w := helpers.TestRequestAsUser(r, "DELETE", "/v1/vehicle/"+testVehicle[0].(string), authUUID, nil)
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, w.Code)
	}

	w = helpers.TestRequestAsUser(r, "GET", "/v1/vehicle/"+testVehicle[0].(string), authUUID, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d for deleted vehicle, got %d", http.StatusNotFound, w.Code)
	}

	if err := a.Gorm.First(&models.Vehicle{}, "id = ?", testVehicle[0].(string)).Error; err != gorm.ErrRecordNotFound {
		t.Error("Expected vehicle to be deleted, but found it in the database")
	}

	if err := a.Gorm.First(&models.MaintenanceRecord{}, "vehicle_id = ?", testVehicle[0].(string)).Error; err != gorm.ErrRecordNotFound {
		t.Error("Expected no maintenance records after vehicle deletion, but found some")
	}

	if err := a.Gorm.First(&models.VehicleUserAccess{}, "vehicle_id = ?", testVehicle[0].(string)).Error; err != gorm.ErrRecordNotFound {
		t.Error("Expected no vehicle user access records after deletion, but found some")
	}

}

func TestVehicleReadPermissions(t *testing.T) {
	r, _ := setupTest(t)

	var successMatrix [8][5]string

	for userIndex, accessRow := range VehicleAccessMatrix {
		for vehicleIndex, access := range accessRow {
			authUUID := AllUsers[userIndex][0]
			vehicle := AllVehicles[vehicleIndex]
			expected := loadVehicle(vehicleIndex)

			w := helpers.TestRequestAsUser(r, "GET", "/v1/vehicle/"+vehicle[0].(string), authUUID, nil)
			if w.Code == http.StatusOK {
				if access == NO_ACCESS {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusNotFound, w.Code)
					continue
				}
				var vehicleResponse models.VehicleBase
				if err := json.Unmarshal(w.Body.Bytes(), &vehicleResponse); err != nil {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Failed to unmarshal response: %v", err)
					continue
				}
				if errMsg := validateVehicleResponse(vehicleResponse, expected); errMsg != "" {
					successMatrix[userIndex][vehicleIndex] = errMsg
					continue
				}

			} else if w.Code == http.StatusNotFound {
				if access > NO_ACCESS {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusOK, w.Code)
					continue
				}
			} else {
				successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Unexpected status code %d for %s reading %s", w.Code, authUUID, vehicle[0].(string))
				continue
			}
		}
	}

	for authUser, access := range successMatrix {
		for readUser, errorMessage := range access {
			if errorMessage != "" {
				authUserName := AllUsers[authUser][1]
				vehicleName := AllVehicles[readUser][1].(string)
				t.Errorf("Auth User %s reading Vehicle %s: %s", authUserName, vehicleName, errorMessage)
			}
		}
	}
}

func TestVehicleWritePermissions(t *testing.T) {
	r, _ := setupTest(t)

	var successMatrix [8][5]string

	for userIndex, accessRow := range VehicleAccessMatrix {
		for vehicleIndex, access := range accessRow {
			authUUID := AllUsers[userIndex][0]
			vehicle := AllVehicles[vehicleIndex]
			expected := models.VehicleBase{
				Nickname: "MODIFIED " + vehicle[1].(string),
				Year:     vehicle[2].(int),
				Make:     vehicle[3].(string),
				MakeID:   vehicle[4].(int),
				Model:    vehicle[5].(string),
				ModelID:  vehicle[6].(int),
				Vin:      vehicle[7].(string),
				Lpn:      vehicle[8].(string),
			}
			modified := models.VehicleBase{
				Nickname: expected.Nickname,
			}

			bodyBytes, err := json.Marshal(modified)
			if err != nil {
				t.Fatalf("Failed to marshal modified vehicle: %v", err)
			}
			bodyReader := bytes.NewReader(bodyBytes)

			w := helpers.TestRequestAsUser(r, "PATCH", "/v1/vehicle/"+vehicle[0].(string), authUUID, bodyReader)
			if w.Code == http.StatusOK {
				if access < WRITE {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusNotFound, w.Code)
					continue
				}
				var vehicleResponse models.VehicleBase
				if err := json.Unmarshal(w.Body.Bytes(), &vehicleResponse); err != nil {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Failed to unmarshal response: %v", err)
					continue
				}
				if errMsg := validateVehicleResponse(vehicleResponse, expected); errMsg != "" {
					successMatrix[userIndex][vehicleIndex] = errMsg
					continue
				}

			} else if w.Code == http.StatusNotFound {
				if access > READ_ONLY {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusOK, w.Code)
					continue
				}
			} else if w.Code == http.StatusForbidden {
				if access == WRITE {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusOK, w.Code)
					continue
				}
			} else {
				successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Unexpected status code %d for %s reading %s", w.Code, authUUID, vehicle[0].(string))
				continue
			}
		}
	}

	for authUser, access := range successMatrix {
		for readUser, errorMessage := range access {
			if errorMessage != "" {
				authUserName := AllUsers[authUser][1]
				vehicleName := AllVehicles[readUser][1].(string)
				t.Errorf("Auth User %s writing Vehicle %s: %s", authUserName, vehicleName, errorMessage)
			}
		}
	}
}
