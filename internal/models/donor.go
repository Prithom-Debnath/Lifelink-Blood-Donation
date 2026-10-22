package models

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Donor represents a blood donor document stored in MongoDB.
type Donor struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string        `bson:"name" json:"name"`
	Email        string        `bson:"email" json:"email"`
	Phone        string        `bson:"phone" json:"phone"`
	BloodGroup   string        `bson:"blood_group" json:"blood_group"`
	City         string        `bson:"city" json:"city"`
	Availability bool          `bson:"availability" json:"availability"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// UnmarshalJSON provides flexible deserialization accepting either snake_case (blood_group)
// or camelCase (bloodGroup) from client requests.
func (d *Donor) UnmarshalJSON(data []byte) error {
	type Alias Donor
	aux := &struct {
		BloodGroupCamel string `json:"bloodGroup"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if d.BloodGroup == "" && aux.BloodGroupCamel != "" {
		d.BloodGroup = aux.BloodGroupCamel
	}

	return nil
}

// Validate checks for required fields, valid email format, and supported blood groups.
func (d *Donor) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(d.Email) == "" {
		return errors.New("email is required")
	}
	if !IsValidEmail(d.Email) {
		return errors.New("invalid email format")
	}
	if strings.TrimSpace(d.Phone) == "" {
		return errors.New("phone is required")
	}
	if strings.TrimSpace(d.BloodGroup) == "" {
		return errors.New("blood_group is required")
	}
	if !IsValidBloodGroup(d.BloodGroup) {
		return errors.New("invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-")
	}
	if strings.TrimSpace(d.City) == "" {
		return errors.New("city is required")
	}
	return nil
}
