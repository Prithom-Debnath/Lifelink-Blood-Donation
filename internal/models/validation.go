package models

import (
	"math"
	"regexp"
	"strings"
)

// SupportedBloodGroups lists all standard ABO/Rh blood groups accepted by LifeLink.
var SupportedBloodGroups = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}

// AllowedStatuses lists all valid blood request status values.
var AllowedStatuses = []string{StatusOpen, StatusFulfilled}

// Urgency level constants for blood requests.
const (
	UrgencyNormal   = "normal"
	UrgencyUrgent   = "urgent"
	UrgencyCritical = "critical"
)

// AllowedUrgencies lists all supported urgency levels for blood requests.
var AllowedUrgencies = []string{UrgencyNormal, UrgencyUrgent, UrgencyCritical}

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
	case StatusOpen, StatusFulfilled:
		return true
	default:
		return false
	}
}

// IsValidUrgency checks if an urgency level string is valid (case-insensitive).
func IsValidUrgency(urgency string) bool {
	switch strings.ToLower(strings.TrimSpace(urgency)) {
	case UrgencyNormal, UrgencyUrgent, UrgencyCritical:
		return true
	default:
		return false
	}
}

// NormalizeUrgency converts an urgency string to standard lowercase.
// If empty, defaults to "normal".
func NormalizeUrgency(urgency string) string {
	cleaned := strings.ToLower(strings.TrimSpace(urgency))
	if cleaned == "" {
		return UrgencyNormal
	}
	return cleaned
}

// IsValidCoordinates validates latitude and longitude.
// If both are nil, it is considered valid (coordinates are optional).
// If only one is provided, or values are out of bounds, returns false.
func IsValidCoordinates(lat, lng *float64) bool {
	if lat == nil && lng == nil {
		return true
	}
	if lat == nil || lng == nil {
		return false
	}
	if *lat < -90.0 || *lat > 90.0 {
		return false
	}
	if *lng < -180.0 || *lng > 180.0 {
		return false
	}
	return true
}

// HaversineDistanceKm calculates the great-circle distance between two geographic coordinates in kilometers.
// Uses the spherical law of haversines with Earth radius = 6371.0 km.
func HaversineDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}
