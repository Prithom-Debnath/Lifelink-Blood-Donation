package models

import (
	"errors"
)

// GetCompatibleDonorBloodGroups returns the list of donor blood groups compatible with the recipient.
//
// DISCLAIMER: This is a simplified red blood cell compatibility matching algorithm intended
// exclusively for educational purposes in CSE 250. It does NOT constitute medical guidance, clinical
// protocol, or advice for actual blood transfusions. Real-world medical transfusions require laboratory
// serological testing, cross-matching, and screening for Rh variants and other minor blood antigens.
func GetCompatibleDonorBloodGroups(recipientGroup string) ([]string, error) {
	if !IsValidBloodGroup(recipientGroup) {
		return nil, errors.New("invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-")
	}

	switch recipientGroup {
	case "AB+":
		// AB+ is the universal recipient for red blood cells
		return []string{"O-", "O+", "B-", "B+", "A-", "A+", "AB-", "AB+"}, nil
	case "AB-":
		return []string{"O-", "B-", "A-", "AB-"}, nil
	case "A+":
		return []string{"O-", "O+", "A-", "A+"}, nil
	case "A-":
		return []string{"O-", "A-"}, nil
	case "B+":
		return []string{"O-", "O+", "B-", "B+"}, nil
	case "B-":
		return []string{"O-", "B-"}, nil
	case "O+":
		return []string{"O-", "O+"}, nil
	case "O-":
		// O- can only safely receive red blood cells from O-
		return []string{"O-"}, nil
	default:
		return nil, errors.New("invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-")
	}
}

// IsBloodCompatible checks whether a donor with donorGroup can safely donate red blood cells to recipientGroup
// according to standard ABO/Rh compatibility rules.
//
// DISCLAIMER: This educational function is not medical advice.
func IsBloodCompatible(donorGroup, recipientGroup string) bool {
	compatibleGroups, err := GetCompatibleDonorBloodGroups(recipientGroup)
	if err != nil {
		return false
	}

	for _, bg := range compatibleGroups {
		if bg == donorGroup {
			return true
		}
	}

	return false
}
