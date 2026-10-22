package models

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Allowed blood request status values.
const (
	StatusOpen      = "open"
	StatusFulfilled = "fulfilled"
	StatusCancelled = "cancelled"
)

// BloodRequest represents an urgent patient request for blood donation stored in MongoDB.
type BloodRequest struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	PatientName  string        `bson:"patient_name" json:"patient_name"`
	BloodGroup   string        `bson:"blood_group" json:"blood_group"`
	Hospital     string        `bson:"hospital" json:"hospital"`
	City         string        `bson:"city" json:"city"`
	ContactPhone string        `bson:"contact_phone" json:"contact_phone"`
	Status       string        `bson:"status" json:"status"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// UnmarshalJSON provides flexible deserialization accepting either snake_case
// (e.g. patient_name, blood_group, contact_phone) or camelCase (e.g. patientName, bloodGroup, contactPhone).
func (r *BloodRequest) UnmarshalJSON(data []byte) error {
	type Alias BloodRequest
	aux := &struct {
		PatientNameCamel  string `json:"patientName"`
		BloodGroupCamel   string `json:"bloodGroup"`
		ContactPhoneCamel string `json:"contactPhone"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if r.PatientName == "" && aux.PatientNameCamel != "" {
		r.PatientName = aux.PatientNameCamel
	}
	if r.BloodGroup == "" && aux.BloodGroupCamel != "" {
		r.BloodGroup = aux.BloodGroupCamel
	}
	if r.ContactPhone == "" && aux.ContactPhoneCamel != "" {
		r.ContactPhone = aux.ContactPhoneCamel
	}

	return nil
}

// Validate checks for required fields, supported blood groups, and allowed statuses.
func (r *BloodRequest) Validate() error {
	if strings.TrimSpace(r.PatientName) == "" {
		return errors.New("patient_name is required")
	}
	if strings.TrimSpace(r.BloodGroup) == "" {
		return errors.New("blood_group is required")
	}
	if !IsValidBloodGroup(r.BloodGroup) {
		return errors.New("invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-")
	}
	if strings.TrimSpace(r.Hospital) == "" {
		return errors.New("hospital is required")
	}
	if strings.TrimSpace(r.City) == "" {
		return errors.New("city is required")
	}
	if strings.TrimSpace(r.ContactPhone) == "" {
		return errors.New("contact_phone is required")
	}
	if strings.TrimSpace(r.Status) != "" && !IsValidStatus(r.Status) {
		return errors.New("invalid status: must be one of open, fulfilled, cancelled")
	}
	return nil
}
