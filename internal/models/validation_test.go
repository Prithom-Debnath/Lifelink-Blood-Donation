package models_test

import (
	"testing"

	"lifelink/internal/models"
)

func TestValidation_IsValidBloodGroup(t *testing.T) {
	validGroups := []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}
	for _, bg := range validGroups {
		if !models.IsValidBloodGroup(bg) {
			t.Errorf("Expected blood group '%s' to be valid", bg)
		}
	}

	invalidGroups := []string{"", " ", "C+", "O", "AB", "A++", "X-", "positive", "123"}
	for _, bg := range invalidGroups {
		if models.IsValidBloodGroup(bg) {
			t.Errorf("Expected blood group '%s' to be invalid", bg)
		}
	}
}

func TestValidation_IsValidEmail(t *testing.T) {
	validEmails := []string{
		"donor@example.com",
		"nusrat.jahan@domain.org",
		"user+tag@gmail.com",
		"name123@hospital.org.bd",
	}
	for _, email := range validEmails {
		if !models.IsValidEmail(email) {
			t.Errorf("Expected email '%s' to be valid", email)
		}
	}

	invalidEmails := []string{
		"",
		"   ",
		"plainaddress",
		"@missingusername.com",
		"username@",
		"username@domain",
		"username@domain.",
		"username@domain.c",
		"username@.com",
	}
	for _, email := range invalidEmails {
		if models.IsValidEmail(email) {
			t.Errorf("Expected email '%s' to be invalid", email)
		}
	}
}

func TestValidation_IsValidStatus(t *testing.T) {
	validStatuses := []string{models.StatusOpen, models.StatusFulfilled}
	for _, s := range validStatuses {
		if !models.IsValidStatus(s) {
			t.Errorf("Expected status '%s' to be valid", s)
		}
	}

	invalidStatuses := []string{"", "pending", "closed", "cancelled", "in_progress", "OPEN", "123"}
	for _, s := range invalidStatuses {
		if models.IsValidStatus(s) {
			t.Errorf("Expected status '%s' to be invalid", s)
		}
	}
}

func TestDonor_Validate(t *testing.T) {
	baseValid := func() models.Donor {
		return models.Donor{
			Name:         "Sadia Islam",
			Email:        "sadia@example.com",
			Phone:        "+8801700112233",
			BloodGroup:   "O+",
			City:         "Dhaka",
			Availability: true,
		}
	}

	// 1. Valid donor should return no error
	valid := baseValid()
	if err := valid.Validate(); err != nil {
		t.Fatalf("Expected valid donor to pass validation, got: %v", err)
	}

	// 2. Missing name
	d := baseValid()
	d.Name = ""
	if err := d.Validate(); err == nil || err.Error() != "name is required" {
		t.Errorf("Expected 'name is required', got: %v", err)
	}

	// 3. Whitespace name
	d = baseValid()
	d.Name = "   "
	if err := d.Validate(); err == nil || err.Error() != "name is required" {
		t.Errorf("Expected 'name is required', got: %v", err)
	}

	// 4. Missing email
	d = baseValid()
	d.Email = ""
	if err := d.Validate(); err == nil || err.Error() != "email is required" {
		t.Errorf("Expected 'email is required', got: %v", err)
	}

	// 5. Invalid email format
	d = baseValid()
	d.Email = "notanemail"
	if err := d.Validate(); err == nil || err.Error() != "invalid email format" {
		t.Errorf("Expected 'invalid email format', got: %v", err)
	}

	// 6. Missing phone
	d = baseValid()
	d.Phone = ""
	if err := d.Validate(); err == nil || err.Error() != "phone is required" {
		t.Errorf("Expected 'phone is required', got: %v", err)
	}

	// 7. Missing blood group
	d = baseValid()
	d.BloodGroup = ""
	if err := d.Validate(); err == nil || err.Error() != "blood_group is required" {
		t.Errorf("Expected 'blood_group is required', got: %v", err)
	}

	// 8. Unsupported blood group
	d = baseValid()
	d.BloodGroup = "Z+"
	if err := d.Validate(); err == nil || err.Error() != "invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-" {
		t.Errorf("Expected unsupported blood group error, got: %v", err)
	}

	// 9. Missing city
	d = baseValid()
	d.City = ""
	if err := d.Validate(); err == nil || err.Error() != "city is required" {
		t.Errorf("Expected 'city is required', got: %v", err)
	}
}

func TestBloodRequest_Validate(t *testing.T) {
	baseValid := func() models.BloodRequest {
		return models.BloodRequest{
			PatientName:  "Arman Khan",
			BloodGroup:   "AB-",
			Hospital:     "Chittagong Medical College",
			City:         "Chittagong",
			ContactPhone: "+8801811223344",
		}
	}

	// 1. Valid request with empty status (allowed, defaults to open on create)
	req := baseValid()
	if err := req.Validate(); err != nil {
		t.Fatalf("Expected valid blood request to pass validation, got: %v", err)
	}

	// 2. Valid request with explicit statuses
	for _, status := range []string{models.StatusOpen, models.StatusFulfilled} {
		req = baseValid()
		req.Status = status
		if err := req.Validate(); err != nil {
			t.Errorf("Expected status '%s' to be valid, got: %v", status, err)
		}
	}

	// 3. Missing patient name
	req = baseValid()
	req.PatientName = "   "
	if err := req.Validate(); err == nil || err.Error() != "patient_name is required" {
		t.Errorf("Expected 'patient_name is required', got: %v", err)
	}

	// 4. Missing blood group
	req = baseValid()
	req.BloodGroup = ""
	if err := req.Validate(); err == nil || err.Error() != "blood_group is required" {
		t.Errorf("Expected 'blood_group is required', got: %v", err)
	}

	// 5. Unsupported blood group
	req = baseValid()
	req.BloodGroup = "C-"
	if err := req.Validate(); err == nil || err.Error() != "invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-" {
		t.Errorf("Expected unsupported blood group error, got: %v", err)
	}

	// 6. Missing hospital
	req = baseValid()
	req.Hospital = ""
	if err := req.Validate(); err == nil || err.Error() != "hospital is required" {
		t.Errorf("Expected 'hospital is required', got: %v", err)
	}

	// 7. Missing city
	req = baseValid()
	req.City = ""
	if err := req.Validate(); err == nil || err.Error() != "city is required" {
		t.Errorf("Expected 'city is required', got: %v", err)
	}

	// 8. Missing contact phone
	req = baseValid()
	req.ContactPhone = ""
	if err := req.Validate(); err == nil || err.Error() != "contact_phone is required" {
		t.Errorf("Expected 'contact_phone is required', got: %v", err)
	}

	// 9. Invalid status
	req = baseValid()
	req.Status = "in_progress"
	if err := req.Validate(); err == nil || err.Error() != "invalid status: must be one of open or fulfilled" {
		t.Errorf("Expected invalid status error, got: %v", err)
	}
}

func TestIsValidUrgency(t *testing.T) {
	valid := []string{"normal", "urgent", "critical", "Normal", "URGENT", "Critical"}
	for _, u := range valid {
		if !models.IsValidUrgency(u) {
			t.Errorf("Expected urgency '%s' to be valid", u)
		}
	}

	invalid := []string{"low", "high", "asap", "", " "}
	for _, u := range invalid {
		if models.IsValidUrgency(u) {
			t.Errorf("Expected urgency '%s' to be invalid", u)
		}
	}

	if models.NormalizeUrgency("") != models.UrgencyNormal {
		t.Errorf("Expected empty urgency to normalize to 'normal'")
	}
	if models.NormalizeUrgency("Critical") != "critical" {
		t.Errorf("Expected 'Critical' to normalize to 'critical'")
	}
}

func TestIsValidCoordinates(t *testing.T) {
	lat := 23.8103
	lng := 90.4125
	invalidLat := 95.0
	invalidLng := 190.0

	// Both nil is valid (optional)
	if !models.IsValidCoordinates(nil, nil) {
		t.Errorf("Expected (nil, nil) coordinates to be valid")
	}

	// Both valid
	if !models.IsValidCoordinates(&lat, &lng) {
		t.Errorf("Expected valid coordinates to pass")
	}

	// Only one nil
	if models.IsValidCoordinates(&lat, nil) {
		t.Errorf("Expected (lat, nil) to be invalid")
	}
	if models.IsValidCoordinates(nil, &lng) {
		t.Errorf("Expected (nil, lng) to be invalid")
	}

	// Out of bounds
	if models.IsValidCoordinates(&invalidLat, &lng) {
		t.Errorf("Expected out of bound lat to be invalid")
	}
	if models.IsValidCoordinates(&lat, &invalidLng) {
		t.Errorf("Expected out of bound lng to be invalid")
	}
}

func TestHaversineDistanceKm(t *testing.T) {
	// Same point distance should be 0
	d0 := models.HaversineDistanceKm(23.8103, 90.4125, 23.8103, 90.4125)
	if d0 > 0.001 {
		t.Errorf("Expected 0 distance for identical coordinates, got %f", d0)
	}

	// Dhaka (23.8103, 90.4125) to Chittagong (22.3569, 91.7832) is approx 210-220 km
	d1 := models.HaversineDistanceKm(23.8103, 90.4125, 22.3569, 91.7832)
	if d1 < 200 || d1 > 230 {
		t.Errorf("Expected distance between Dhaka and Chittagong to be ~215 km, got %f", d1)
	}

	// Dhanmondi (23.7465, 90.3760) to Gulshan (23.7925, 90.4078) is approx 6-7 km
	d2 := models.HaversineDistanceKm(23.7465, 90.3760, 23.7925, 90.4078)
	if d2 < 5 || d2 > 8 {
		t.Errorf("Expected distance between Dhanmondi and Gulshan to be ~6.5 km, got %f", d2)
	}
}
