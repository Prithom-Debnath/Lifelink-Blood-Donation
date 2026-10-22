package models_test

import (
	"encoding/json"
	"testing"

	"lifelink/internal/models"
)

func TestBloodRequest_UnmarshalJSON(t *testing.T) {
	// Case 1: snake_case JSON with urgency and coordinates
	jsonSnake := []byte(`{
		"patient_name": "Farhana Yasmin",
		"blood_group": "B+",
		"hospital": "Dhaka Medical College Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801912345678",
		"status": "open",
		"urgency": "critical",
		"latitude": 23.7258,
		"longitude": 90.3976
	}`)

	var req1 models.BloodRequest
	if err := json.Unmarshal(jsonSnake, &req1); err != nil {
		t.Fatalf("Failed to unmarshal snake_case JSON: %v", err)
	}

	if req1.PatientName != "Farhana Yasmin" || req1.BloodGroup != "B+" || req1.Status != models.StatusOpen {
		t.Errorf("Mismatch in snake_case unmarshal: %+v", req1)
	}
	if req1.Urgency != "critical" {
		t.Errorf("Expected urgency 'critical', got '%s'", req1.Urgency)
	}
	if req1.Latitude == nil || *req1.Latitude != 23.7258 {
		t.Errorf("Expected latitude 23.7258, got %v", req1.Latitude)
	}
	if req1.Longitude == nil || *req1.Longitude != 90.3976 {
		t.Errorf("Expected longitude 90.3976, got %v", req1.Longitude)
	}

	// Case 2: camelCase JSON
	jsonCamel := []byte(`{
		"patientName": "Kamrul Islam",
		"bloodGroup": "O-",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contactPhone": "+8801711122233",
		"status": "fulfilled",
		"urgency": "urgent"
	}`)

	var req2 models.BloodRequest
	if err := json.Unmarshal(jsonCamel, &req2); err != nil {
		t.Fatalf("Failed to unmarshal camelCase JSON: %v", err)
	}

	if req2.PatientName != "Kamrul Islam" || req2.BloodGroup != "O-" || req2.Status != models.StatusFulfilled {
		t.Errorf("Mismatch in camelCase unmarshal: %+v", req2)
	}
	if req2.Urgency != "urgent" {
		t.Errorf("Expected urgency 'urgent', got '%s'", req2.Urgency)
	}
}

func TestBloodRequest_UrgencyAndCoordinatesValidation(t *testing.T) {
	lat := 23.8103
	lng := 90.4125
	invalidLat := 100.0

	req := models.BloodRequest{
		PatientName:  "Test Patient",
		BloodGroup:   "A+",
		Hospital:     "Hospital X",
		City:         "Dhaka",
		ContactPhone: "+8801711000000",
		Status:       "open",
		Urgency:      "critical",
		Latitude:     &lat,
		Longitude:    &lng,
	}

	// Valid
	if err := req.Validate(); err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}

	// Invalid urgency
	reqInvalidUrgency := req
	reqInvalidUrgency.Urgency = "extremely-urgent"
	if err := reqInvalidUrgency.Validate(); err == nil {
		t.Errorf("Expected error for invalid urgency, got nil")
	}

	// Valid with different cases
	reqUpperUrgency := req
	reqUpperUrgency.Urgency = "Urgent"
	if err := reqUpperUrgency.Validate(); err != nil {
		t.Errorf("Expected 'Urgent' to be valid, got: %v", err)
	}

	// Invalid latitude
	reqInvalidLat := req
	reqInvalidLat.Latitude = &invalidLat
	if err := reqInvalidLat.Validate(); err == nil {
		t.Errorf("Expected error for latitude out of range, got nil")
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
