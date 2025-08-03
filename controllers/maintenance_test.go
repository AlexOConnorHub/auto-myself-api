package controllers

import (
	"auto-myself-api/helpers"
	"auto-myself-api/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

var AllMaintenances = [][]interface{}{
	{"01978640-1148-74f8-be64-59f2af568e59", 0, 15000, "2023-01-15", "Oil change", "Oil Change", 5000, "miles", "019785fe-4eb4-766e-9c45-bec7780972a2", "$1.32"},
	{"01978640-1148-74f8-be64-5e6b15475861", 0, 20000, "2023-06-20", "Tire rotation", "Tire Rotation", 10000, "miles", "019785fe-4eb4-766e-9c45-bec7780972a2", "$1.32"},
	{"01978640-1148-74f8-be64-600b58c80190", 1, 5000, "2023-02-10", "Brake inspection", "Inspection", 10000, "miles", "019785fe-4eb4-766e-9c45-bec7780972a2", "$1.32"},
	{"01978640-1148-74f8-be64-673c2bc659d3", 1, 12000, "2023-07-05", "Battery check", "Maintenance Check", 20000, "miles", "019785fe-4eb4-766e-9c45-bec7780972a2", "$1.32"},
	{"01978640-1148-74f8-be64-6b2a85c627a7", 1, 8000, "2023-03-12", "Fluid top-up", "Maintenance Check", 5000, "miles", "019785fe-4eb4-766e-9c45-c1f83e7c1f1f", "$1.32"},
	{"01978640-1148-74f8-be64-6ce4acd8abcd", 1, 16000, "2023-08-18", "Tire replacement", "Tire Replacement", 20000, "miles", "019785fe-4eb4-766e-9c45-c1f83e7c1f1f", "$1.32"},
	{"01978640-1148-74f8-be64-70821e946a20", 1, 3000, "2023-04-05", "Oil change", "Oil Change", 5000, "miles", "019785fe-4eb4-766e-9c45-f9cd4ee5c0b3", "$1.32"},
	{"01978640-1148-74f8-be64-74b319513577", 1, 7000, "2023-09-10", "Brake pad replacement", "Brake Replacement", 10000, "miles", "019785fe-4eb4-766e-9c45-f9cd4ee5c0b3", "$1.32"},
	{"01978640-1148-74f8-be64-7b8ece4dc40f", 1, 1000, "2023-05-15", "Inspection check", "Inspection Check", 5000, "miles", "019785fe-4eb4-766e-9c45-fc6ed4a7407b", "$1.32"},
	{"01978640-1148-74f8-be64-7ea0e41d68d5", 1, 4000, "2023-10-20", "Tire rotation and balance", "Tire Rotation and Balance", 10000, "miles", "019785fe-4eb4-766e-9c45-fc6ed4a7407b", "$1.32"},
	{"01978640-1148-74f8-be64-bc7c09adc4c1", 2, 3000, "2023-04-05", "Oil change", "Oil Change", 5000, "miles", "019785fe-4eb4-766e-9c45-c8578456b4df", "$1.32"},
	{"01978640-1148-74f8-be64-ac02bbbf11cf", 2, 7000, "2023-09-10", "Brake pad replacement", "Brake Replacement", 10000, "miles", "019785fe-4eb4-766e-9c45-c8578456b4df", "$1.32"},
	{"01978640-1148-74f8-be64-b446e827f938", 3, 3000, "2023-04-05", "Oil change", "Oil Change", 5000, "miles", "019785fe-4eb4-766e-9c45-f592a1187d0c", "$1.32"},
	{"01978640-1148-74f8-be64-ba806fa103c7", 3, 7000, "2023-09-10", "Brake pad replacement", "Brake Replacement", 10000, "miles", "019785fe-4eb4-766e-9c45-f592a1187d0c", "$1.32"},
	{"01978640-1149-7118-bada-9f77b4fa870a", 3, 2000, "2023-03-01", "Oil change", "Oil Change", 5000, "miles", "019785fe-4eb4-766e-9c45-f9cd4ee5c0b3", "$1.32"},
	{"01978640-1149-7118-bada-a16e466c1064", 3, 4000, "2023-08-15", "Tire rotation and balance", "Tire Rotation and Balance", 10000, "miles", "019785fe-4eb4-766e-9c45-f9cd4ee5c0b3", "$1.32"},
	{"01978640-1149-7118-bada-a7ce46886414", 4, 5000, "2023-02-10", "Brake inspection", "Inspection", 10000, "miles", "019785fe-4eb4-766e-9c45-fc6ed4a7407b", "$1.32"},
	{"01978640-1149-7118-bada-aa596985d112", 4, 12000, "2023-07-05", "Battery check", "Maintenance Check", 20000, "miles", "019785fe-4eb4-766e-9c45-fc6ed4a7407b", "$1.32"},
}

func validateMaintenanceResponse(marshaledResponse models.MaintenanceRecordBase, expected models.MaintenanceRecordBase) string {
	if marshaledResponse.VehicleID != expected.VehicleID {
		return fmt.Sprintf("Expected vehicleID to be %s, got %s", expected.VehicleID, marshaledResponse.VehicleID)
	}
	if marshaledResponse.Timestamp.Format("2006-01-02") != expected.Timestamp.Format("2006-01-02") {
		return fmt.Sprintf("Expected timestamp to be %s, got %s", expected.Timestamp, marshaledResponse.Timestamp)
	}
	if marshaledResponse.Odometer != expected.Odometer {
		return fmt.Sprintf("Expected odometer to be %d, got %d", expected.Odometer, marshaledResponse.Odometer)
	}
	if marshaledResponse.Notes != expected.Notes {
		return fmt.Sprintf("Expected notes to be %s, got %s", expected.Notes, marshaledResponse.Notes)
	}
	if marshaledResponse.Type != expected.Type {
		return fmt.Sprintf("Expected type to be %s, got %s", expected.Type, marshaledResponse.Type)
	}
	if marshaledResponse.Interval != expected.Interval {
		return fmt.Sprintf("Expected interval to be %d, got %d", expected.Interval, marshaledResponse.Interval)
	}
	if marshaledResponse.IntervalType != expected.IntervalType {
		return fmt.Sprintf("Expected intervalType to be %s, got %s", expected.IntervalType, marshaledResponse.IntervalType)
	}
	if marshaledResponse.Cost != expected.Cost {
		return fmt.Sprintf("Expected cost to be %s, got %s", expected.Cost, marshaledResponse.Cost)
	}
	return ""
}

func loadMaintenanceRecord(index int) models.MaintenanceRecordBase {
	maintenance := AllMaintenances[index]
	vehicleID, err := uuid.FromString(AllVehicles[maintenance[1].(int)][0].(string))
	if err != nil {
		panic(fmt.Sprintf("Failed to parse vehicle ID: %v (index %d)", err, index))
	}

	timestampStr := maintenance[3].(string)
	timestamp, err := time.Parse("2006-01-02", timestampStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse timestamp: %v", err))
	}

	return models.MaintenanceRecordBase{
		VehicleID:    vehicleID,
		Timestamp:    timestamp,
		Odometer:     maintenance[2].(int),
		Notes:        maintenance[4].(string),
		Type:         maintenance[5].(string),
		Interval:     maintenance[6].(int),
		IntervalType: maintenance[7].(string),
		Cost:         maintenance[9].(string),
	}
}

func TestMaintenancesVerifyAllExist(t *testing.T) {
	r := setupTest(t)

	for index, maintenanceRecord := range AllMaintenances {
		expected := loadMaintenanceRecord(index)
		vehicleOwnerUUID := AllVehicles[maintenanceRecord[1].(int)][9].(string)
		w := helpers.PerformRequest(r, "GET", "/maintenance/"+maintenanceRecord[0].(string), map[string]string{"auth_uuid": vehicleOwnerUUID, "content-type": "application/json"}, nil)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
			continue
		}
		var response models.MaintenanceRecordBase
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Errorf("Failed to unmarshal response: %v", err)
			continue
		}
		if errMsg := validateMaintenanceResponse(response, expected); errMsg != "" {
			t.Error(errMsg)
			continue
		}
	}
}

func TestMaintenanceList(t *testing.T) {
	r := setupTest(t)

	maintenanceCountHash := make(map[int]int)
	for _, maintenanceRow := range AllMaintenances {
		vehicleID := maintenanceRow[1].(int)
		maintenanceCountHash[vehicleID]++
	}

	for userIndex, userAccess := range VehicleAccessMatrix {
		for vehicleIndex, vehicleRow := range AllVehicles {
			if userAccess[vehicleIndex] < READ_ONLY {
				continue
			}
			auth_uuid := AllUsers[userIndex][0]
			vehicleID := vehicleRow[0].(string)

			w := helpers.PerformRequest(r, "GET", "/vehicle/"+vehicleID+"/maintenance", map[string]string{"auth_uuid": auth_uuid, "content-type": "application/json"}, nil)
			if w.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d for user %s", http.StatusOK, w.Code, AllUsers[userIndex][1])
				continue
			}

			var locations []string
			if err := json.Unmarshal(w.Body.Bytes(), &locations); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
			}

			if len(locations) != maintenanceCountHash[vehicleIndex] {
				t.Errorf("Expected %d maintenance records, got %d for user %s", maintenanceCountHash[vehicleIndex], len(locations), AllUsers[userIndex][1])
			}
		}
	}
}

func TestMaintenancePost(t *testing.T) {
	r := setupTest(t)

	vehicleUUID, err := uuid.FromString(AllVehicles[0][0].(string))
	if err != nil {
		t.Fatalf("Failed to parse vehicle UUID: %v", err)
	}

	newMaintenance := models.MaintenanceRecordBase{
		VehicleID:    vehicleUUID,
		Timestamp:    time.Now(),
		Odometer:     10000,
		Notes:        "TEST Initial maintenance",
		Type:         "TEST Oil Change",
		Interval:     5000,
		IntervalType: "miles",
		Cost:         "100.00",
	}

	bodyBytes, err := json.Marshal(newMaintenance)
	if err != nil {
		t.Fatalf("Failed to marshal modified maintenance record: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w := helpers.PerformRequest(r, "POST", "/maintenance", map[string]string{"auth_uuid": AllVehicles[0][9].(string), "content-type": "application/json"}, bodyReader)
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	location := w.Header().Get("X-Object-Location")
	if location == "" {
		t.Error("Expected X-Object-Location header to be set, but it was empty")
	}

	w = helpers.PerformRequest(r, "GET", location, map[string]string{"auth_uuid": AllVehicles[0][9].(string), "content-type": "application/json"}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response models.MaintenanceRecordBase
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if errMsg := validateMaintenanceResponse(response, newMaintenance); errMsg != "" {
		t.Error(errMsg)
	}
}

func TestMaintenancePatch(t *testing.T) {
	r := setupTest(t)

	expected := loadMaintenanceRecord(0)
	expected.Notes += " MODIFIED"
	modified := models.MaintenanceRecordBase{
		Notes: expected.Notes,
	}

	bodyBytes, err := json.Marshal(modified)
	if err != nil {
		t.Fatalf("Failed to marshal modified maintenance record: %v", err)
	}
	bodyReader := bytes.NewReader(bodyBytes)

	w := helpers.PerformRequest(r, "PATCH", "/maintenance/"+AllMaintenances[0][0].(string), map[string]string{"auth_uuid": AllVehicles[AllMaintenances[0][1].(int)][9].(string), "content-type": "application/json"}, bodyReader)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response models.MaintenanceRecordBase
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if errMsg := validateMaintenanceResponse(response, expected); errMsg != "" {
		t.Error(errMsg)
	}
}

func TestMaintenanceDelete(t *testing.T) {
	r := setupTest(t)

	maintenanceUUID := AllMaintenances[0][0].(string)
	w := helpers.PerformRequest(r, "DELETE", "/maintenance/"+maintenanceUUID, map[string]string{"auth_uuid": AllVehicles[AllMaintenances[0][1].(int)][9].(string), "content-type": "application/json"}, nil)
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status code %d, got %d", http.StatusNoContent, w.Code)
	}

	w = helpers.PerformRequest(r, "GET", "/maintenance/"+maintenanceUUID, map[string]string{"auth_uuid": AllVehicles[AllMaintenances[0][1].(int)][9].(string), "content-type": "application/json"}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status code %d after deletion, got %d", http.StatusNotFound, w.Code)
	}
}

func TestMaintenanceReadPermissions(t *testing.T) {
	r := setupTest(t)

	var errors []string

	for maintenanceRowIndex, maintenanceRow := range AllMaintenances {
		maintenance := loadMaintenanceRecord(maintenanceRowIndex)
		for userIndex, userRow := range AllUsers {
			authUUID := userRow[0]
			canRead := VehicleAccessMatrix[userIndex][maintenanceRow[1].(int)] >= READ_ONLY

			w := helpers.PerformRequest(r, "GET", "/maintenance/"+maintenanceRow[0].(string), map[string]string{"auth_uuid": authUUID, "content-type": "application/json"}, nil)
			if w.Code == http.StatusOK {
				if !canRead {
					errors = append(errors, fmt.Sprintf("User %s should not have read access to maintenance %s but got status %d", authUUID, maintenanceRow[0], w.Code))
					continue
				}
				var response models.MaintenanceRecordBase
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to unmarshal response for user %s: %v", authUUID, err))
					continue
				}
				if errMsg := validateMaintenanceResponse(response, maintenance); errMsg != "" {
					errors = append(errors, fmt.Sprintf("Validation error for user %s on maintenance %s: %s", authUUID, maintenanceRow[0], errMsg))
				}
			} else if w.Code == http.StatusNotFound {
				if canRead {
					errors = append(errors, fmt.Sprintf("User %s should have read access to maintenance %s but got status %d", authUUID, maintenanceRow[0], w.Code))
				}
			} else {
				errors = append(errors, fmt.Sprintf("Unexpected status code %d for user %s reading maintenance %s", w.Code, authUUID, maintenanceRow[0]))
			}

		}
	}

	for _, errorMessage := range errors {
		t.Error(errorMessage)
	}
}

func TestMaintenanceWritePermissions(t *testing.T) {
	r := setupTest(t)

	var errors []string

	for maintenanceRowIndex, maintenanceRow := range AllMaintenances {
		maintenance := loadMaintenanceRecord(maintenanceRowIndex)
		for userIndex, userRow := range AllUsers {
			authUUID := userRow[0]
			permission := VehicleAccessMatrix[userIndex][maintenanceRow[1].(int)]

			modified := models.MaintenanceRecordBase{
				Notes: maintenance.Notes + " MODIFIED",
			}
			maintenance.Notes = modified.Notes
			bodyBytes, err := json.Marshal(modified)
			if err != nil {
				t.Fatalf("Failed to marshal modified maintenance record: %v", err)
			}
			bodyReader := bytes.NewReader(bodyBytes)

			w := helpers.PerformRequest(r, "PATCH", "/maintenance/"+maintenanceRow[0].(string), map[string]string{"auth_uuid": authUUID, "content-type": "application/json"}, bodyReader)
			if w.Code == http.StatusOK {
				if permission < WRITE {
					errors = append(errors, fmt.Sprintf("User %s should not have write access to maintenance %s but got status %d", userRow[1], maintenanceRow[4], w.Code))
					continue
				}
				var response models.MaintenanceRecordBase
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to unmarshal response for user %s and maintenance %s: %v", userRow[1], maintenanceRow[4], err))
					continue
				}
				if errMsg := validateMaintenanceResponse(response, maintenance); errMsg != "" {
					errors = append(errors, fmt.Sprintf("Validation error for user %s on maintenance %s: %s", userRow[1], maintenanceRow[4], errMsg))
				}
			} else if w.Code == http.StatusNotFound {
				if permission == WRITE {
					errors = append(errors, fmt.Sprintf("User %s should have write access to maintenance %s but got status %d", userRow[1], maintenanceRow[4], w.Code))
				} else if permission == READ_ONLY {
					errors = append(errors, fmt.Sprintf("User %s should not get 404 to maintenance %s but got status %d", userRow[1], maintenanceRow[4], w.Code))
				}
			} else if w.Code == http.StatusForbidden {
				if permission == WRITE {
					errors = append(errors, fmt.Sprintf("User %s should have write access to maintenance %s but got status %d", userRow[1], maintenanceRow[4], w.Code))
				} else if permission == NO_ACCESS {
					errors = append(errors, fmt.Sprintf("User %s should not not get 403 to maintenance %s but got status %d", userRow[1], maintenanceRow[4], w.Code))
				}
			} else {
				errors = append(errors, fmt.Sprintf("Unexpected status code %d for user %s reading maintenance %s", w.Code, userRow[1], maintenanceRow[4]))
			}

		}
	}

	for _, errorMessage := range errors {
		t.Error(errorMessage)
	}
}
