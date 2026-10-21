package models_test

import (
	"encoding/json"
	"testing"

	"lifelink/internal/models"
)

func TestBloodRequest_UnmarshalJSON(t *testing.T) {
	// Case 1: snake_case JSON
	jsonSnake := []byte(`{
		"patient_name": "Farhana Yasmin",
		"blood_group": "B+",
		"hospital": "Dhaka Medical College Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801912345678",
		"status": "open"
	}`)

	var req1 models.BloodRequest
	if err := json.Unmarshal(jsonSnake, &req1); err != nil {
		t.Fatalf("Failed to unmarshal snake_case JSON: %v", err)
	}

	if req1.PatientName != "Farhana Yasmin" || req1.BloodGroup != "B+" || req1.Status != models.StatusOpen {
		t.Errorf("Mismatch in snake_case unmarshal: %+v", req1)
	}

	// Case 2: camelCase JSON
	jsonCamel := []byte(`{
		"patientName": "Kamrul Islam",
		"bloodGroup": "O-",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contactPhone": "+8801711122233",
		"status": "fulfilled"
	}`)

	var req2 models.BloodRequest
	if err := json.Unmarshal(jsonCamel, &req2); err != nil {
		t.Fatalf("Failed to unmarshal camelCase JSON: %v", err)
	}

	if req2.PatientName != "Kamrul Islam" || req2.BloodGroup != "O-" || req2.Status != models.StatusFulfilled {
		t.Errorf("Mismatch in camelCase unmarshal: %+v", req2)
	}
}

func TestBloodRequest_StatusConstants(t *testing.T) {
	if models.StatusOpen != "open" {
		t.Errorf("Expected StatusOpen to be 'open', got %s", models.StatusOpen)
	}
	if models.StatusFulfilled != "fulfilled" {
		t.Errorf("Expected StatusFulfilled to be 'fulfilled', got %s", models.StatusFulfilled)
	}
	if models.StatusCancelled != "cancelled" {
		t.Errorf("Expected StatusCancelled to be 'cancelled', got %s", models.StatusCancelled)
	}
}
