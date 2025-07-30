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

var allVehicles = [][]interface{}{
	{"019785fe-4eb4-766e-9c45-d0b2bb289b82", "Vehicle 1", 2015, "MAZDA", 473, "MX-5", 2072, "VIN CAR 1", "LPN CAR 1", "019785fe-4eb4-766e-9c45-bec7780972a2"},
	{"019785fe-4eb4-766e-9c45-d77f41aa8317", "Vehicle 2", 2021, "TOYOTA", 448, "Corolla", 2208, "VIN CAR 2", "LPN CAR 2", "019785fe-4eb4-766e-9c45-bec7780972a2"},
	{"019785fe-4eb4-766e-9c45-d9cc7ea628c1", "Vehicle 3", 2019, "FORD", 460, "Edge", 1797, "VIN CAR 3", "LPN CAR 3", "019785fe-4eb4-766e-9c45-c8578456b4df"},
	{"019785fe-4eb4-766e-9c45-ddfb4b2e7210", "Vehicle 4", 2018, "NISSAN", 478, "GT-R", 1890, "VIN CAR 4", "LPN CAR 4", "019785fe-4eb4-766e-9c45-f592a1187d0c"},
	{"019785fe-4eb4-766e-9c45-e1af5010246b", "Vehicle 5", 2025, "ASTON MARTIN", 440, "Vanquish", 1701, "VIN CAR 5", "LPN CAR 5", "019785fe-4eb4-766e-9c45-fc6ed4a7407b"},
}

var vehicleAccessMatrix = [8][5]int{
	{WRITE, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, READ_ONLY, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS, NO_ACCESS},
	{NO_ACCESS, NO_ACCESS, NO_ACCESS, WRITE, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, WRITE, NO_ACCESS},
	{NO_ACCESS, WRITE, NO_ACCESS, READ_ONLY, WRITE},
}

func testVehicleResponse(marshaledResponse models.VehicleBase, expected models.VehicleBase) string {
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

func TestVerifyAllVehiclesExist(t *testing.T) {
	r := setupTest(t)

	for _, vehicle := range allVehicles {
		expected := models.VehicleBase{
			Nickname: vehicle[1].(string),
			Year:     vehicle[2].(int),
			Make:     vehicle[3].(string),
			MakeID:   vehicle[4].(int),
			Model:    vehicle[5].(string),
			ModelID:  vehicle[6].(int),
			Vin:      vehicle[7].(string),
			Lpn:      vehicle[8].(string),
		}
		// createdBy := vehicle[9].(str
		w := helpers.PerformRequest(r, "GET", "/vehicle/"+vehicle[0].(string), map[string]string{"auth_uuid": vehicle[9].(string), "content-type": "application/json"}, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			continue
		}
		var vehicle models.VehicleBase
		if err := json.Unmarshal(w.Body.Bytes(), &vehicle); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			continue
		}
		if errMsg := testVehicleResponse(vehicle, expected); errMsg != "" {
			t.Error(errMsg)
			continue
		}
	}
}

func TestModifyVehicle(t *testing.T) {
	r := setupTest(t)

	var testVehicle = allVehicles[0]
	auth_uuid := testVehicle[9].(string)
	expected := models.VehicleBase{
		Nickname: "MODIFIED " + testVehicle[1].(string),
		Year:     testVehicle[2].(int),
		Make:     testVehicle[3].(string),
		MakeID:   testVehicle[4].(int),
		Model:    testVehicle[5].(string),
		ModelID:  testVehicle[6].(int),
		Vin:      testVehicle[7].(string),
		Lpn:      testVehicle[8].(string),
	}
	modified := models.VehicleBase{
		Nickname: expected.Nickname,
	}

	bodyBytes, err := json.Marshal(modified)
	if err != nil {
		t.Fatalf("Failed to marshal modified vehicle: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w := helpers.PerformRequest(r, "PATCH", "/vehicle/"+testVehicle[0].(string), map[string]string{"auth_uuid": auth_uuid, "content-type": "application/json"}, bodyReader)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var vehicleResponse models.VehicleBase
	if err := json.Unmarshal(w.Body.Bytes(), &vehicleResponse); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if errMsg := testVehicleResponse(vehicleResponse, expected); errMsg != "" {
		t.Error(errMsg)
	}
}

func TestVehicleReadPermissions(t *testing.T) {
	r := setupTest(t)

	var successMatrix [8][5]string

	for userIndex, accessRow := range vehicleAccessMatrix {
		for vehicleIndex, access := range accessRow {
			auth_uuid := allUsers[userIndex][0]
			vehicle := allVehicles[vehicleIndex]
			expected := models.VehicleBase{
				Nickname: vehicle[1].(string),
				Year:     vehicle[2].(int),
				Make:     vehicle[3].(string),
				MakeID:   vehicle[4].(int),
				Model:    vehicle[5].(string),
				ModelID:  vehicle[6].(int),
				Vin:      vehicle[7].(string),
				Lpn:      vehicle[8].(string),
			}

			w := helpers.PerformRequest(r, "GET", "/vehicle/"+vehicle[0].(string), map[string]string{"auth_uuid": auth_uuid, "content-type": "application/json"}, nil)
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
				if errMsg := testVehicleResponse(vehicleResponse, expected); errMsg != "" {
					successMatrix[userIndex][vehicleIndex] = errMsg
					continue
				}

			} else if w.Code == http.StatusNotFound {
				if access > NO_ACCESS {
					successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Expected %d but got %d", http.StatusOK, w.Code)
					continue
				}
			} else {
				successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Unexpected status code %d for %s reading %s", w.Code, auth_uuid, vehicle[0].(string))
				continue
			}
		}
	}

	for authUser, acces := range successMatrix {
		for readUser, errorMessage := range acces {
			if errorMessage != "" {
				authUserName := allUsers[authUser][1]
				vehicleName := allVehicles[readUser][1].(string)
				t.Errorf("Auth User %s reading Vehicle %s: %s", authUserName, vehicleName, errorMessage)
			}
		}
	}
}

func TestVehicleWritePermissions(t *testing.T) {
	r := setupTest(t)

	var successMatrix [8][5]string

	for userIndex, accessRow := range vehicleAccessMatrix {
		for vehicleIndex, access := range accessRow {
			auth_uuid := allUsers[userIndex][0]
			vehicle := allVehicles[vehicleIndex]
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

			w := helpers.PerformRequest(r, "PATCH", "/vehicle/"+vehicle[0].(string), map[string]string{"auth_uuid": auth_uuid, "content-type": "application/json"}, bodyReader)
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
				if errMsg := testVehicleResponse(vehicleResponse, expected); errMsg != "" {
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
				successMatrix[userIndex][vehicleIndex] = fmt.Sprintf("Unexpected status code %d for %s reading %s", w.Code, auth_uuid, vehicle[0].(string))
				continue
			}
		}
	}

	for authUser, acces := range successMatrix {
		for readUser, errorMessage := range acces {
			if errorMessage != "" {
				authUserName := allUsers[authUser][1]
				vehicleName := allVehicles[readUser][1].(string)
				t.Errorf("Auth User %s writing Vehicle %s: %s", authUserName, vehicleName, errorMessage)
			}
		}
	}
}
