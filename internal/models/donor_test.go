package models_test

import (
	"encoding/json"
	"testing"

	"lifelink/internal/models"
)

func TestDonor_UnmarshalJSON(t *testing.T) {
	// Case 1: snake_case "blood_group"
	jsonSnake := []byte(`{"name":"Ayesha","email":"ayesha@example.com","blood_group":"A+","city":"Chittagong","availability":true}`)
	var d1 models.Donor
	if err := json.Unmarshal(jsonSnake, &d1); err != nil {
		t.Fatalf("Failed to unmarshal snake_case JSON: %v", err)
	}
	if d1.BloodGroup != "A+" || d1.Name != "Ayesha" || !d1.Availability {
		t.Errorf("Mismatch in snake_case unmarshal: %+v", d1)
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
