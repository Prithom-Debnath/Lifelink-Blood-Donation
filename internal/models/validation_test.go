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
	validStatuses := []string{models.StatusOpen, models.StatusFulfilled, models.StatusCancelled}
	for _, s := range validStatuses {
		if !models.IsValidStatus(s) {
			t.Errorf("Expected status '%s' to be valid", s)
		}
	}

	invalidStatuses := []string{"", "pending", "closed", "in_progress", "OPEN", "123"}
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
	for _, status := range []string{models.StatusOpen, models.StatusFulfilled, models.StatusCancelled} {
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
	if err := req.Validate(); err == nil || err.Error() != "invalid status: must be one of open, fulfilled, cancelled" {
		t.Errorf("Expected invalid status error, got: %v", err)
	}
}
