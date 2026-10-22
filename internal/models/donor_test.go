package models_test

import (
	"encoding/json"
	"testing"

	"lifelink/internal/models"
)

func TestDonor_UnmarshalJSON(t *testing.T) {
	// Case 1: snake_case "blood_group" with coordinates
	jsonSnake := []byte(`{"name":"Ayesha","email":"ayesha@example.com","blood_group":"A+","city":"Chittagong","availability":true,"latitude":22.3569,"longitude":91.7832}`)
	var d1 models.Donor
	if err := json.Unmarshal(jsonSnake, &d1); err != nil {
		t.Fatalf("Failed to unmarshal snake_case JSON: %v", err)
	}
	if d1.BloodGroup != "A+" || d1.Name != "Ayesha" || !d1.Availability {
		t.Errorf("Mismatch in snake_case unmarshal: %+v", d1)
	}
	if d1.Latitude == nil || *d1.Latitude != 22.3569 {
		t.Errorf("Expected latitude 22.3569, got %v", d1.Latitude)
	}
	if d1.Longitude == nil || *d1.Longitude != 91.7832 {
		t.Errorf("Expected longitude 91.7832, got %v", d1.Longitude)
	}

	// Case 2: camelCase "bloodGroup"
	jsonCamel := []byte(`{"name":"Karim","email":"karim@example.com","bloodGroup":"B-","city":"Sylhet","availability":false}`)
	var d2 models.Donor
	if err := json.Unmarshal(jsonCamel, &d2); err != nil {
		t.Fatalf("Failed to unmarshal camelCase JSON: %v", err)
	}
	if d2.BloodGroup != "B-" || d2.Name != "Karim" || d2.Availability {
		t.Errorf("Mismatch in camelCase unmarshal: %+v", d2)
	}
}

func TestDonor_CoordinatesValidation(t *testing.T) {
	lat := 23.8103
	lng := 90.4125
	invalidLat := 95.0
	invalidLng := 185.0

	// Valid with coordinates
	dValid := models.Donor{
		Name:         "Valid Donor",
		Email:        "donor@example.com",
		Phone:        "+8801700000000",
		BloodGroup:   "O+",
		City:         "Dhaka",
		Availability: true,
		Latitude:     &lat,
		Longitude:    &lng,
	}
	if err := dValid.Validate(); err != nil {
		t.Errorf("Expected valid donor, got error: %v", err)
	}

	// Invalid latitude
	dInvalidLat := dValid
	dInvalidLat.Latitude = &invalidLat
	if err := dInvalidLat.Validate(); err == nil {
		t.Errorf("Expected error for latitude > 90, got nil")
	}

	// Invalid longitude
	dInvalidLng := dValid
	dInvalidLng.Longitude = &invalidLng
	if err := dInvalidLng.Validate(); err == nil {
		t.Errorf("Expected error for longitude > 180, got nil")
	}

	// Valid without coordinates (nil coordinates are optional)
	dNoCoords := dValid
	dNoCoords.Latitude = nil
	dNoCoords.Longitude = nil
	if err := dNoCoords.Validate(); err != nil {
		t.Errorf("Expected valid donor without coordinates, got error: %v", err)
	}
}
