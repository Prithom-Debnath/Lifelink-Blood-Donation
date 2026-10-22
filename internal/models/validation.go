package models

import (
	"regexp"
	"strings"
)

// SupportedBloodGroups lists all standard ABO/Rh blood groups accepted by LifeLink.
var SupportedBloodGroups = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}

// AllowedStatuses lists all valid blood request status values.
var AllowedStatuses = []string{StatusOpen, StatusFulfilled, StatusCancelled}

// emailRegex is a beginner-friendly regex validating standard email format.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValidBloodGroup checks if a given blood group string is one of the 8 supported ABO/Rh groups.
func IsValidBloodGroup(bg string) bool {
	switch strings.TrimSpace(bg) {
	case "A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-":
		return true
	default:
		return false
	}
}

// IsValidEmail verifies that an email string adheres to standard email formatting.
func IsValidEmail(email string) bool {
	trimmed := strings.TrimSpace(email)
	if len(trimmed) < 3 || len(trimmed) > 254 {
		return false
	}
	return emailRegex.MatchString(trimmed)
}

// IsValidStatus checks if a given blood request status is one of the allowed statuses.
func IsValidStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case StatusOpen, StatusFulfilled, StatusCancelled:
		return true
	default:
		return false
	}
}
