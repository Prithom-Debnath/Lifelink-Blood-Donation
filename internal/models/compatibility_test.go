package models_test

import (
	"testing"

	"lifelink/internal/models"
)

func TestCompatibility_GetCompatibleDonorBloodGroups(t *testing.T) {
	tests := []struct {
		recipient string
		expected  []string
	}{
		{
			recipient: "AB+",
			expected:  []string{"O-", "O+", "B-", "B+", "A-", "A+", "AB-", "AB+"},
		},
		{
			recipient: "AB-",
			expected:  []string{"O-", "B-", "A-", "AB-"},
		},
		{
			recipient: "A+",
			expected:  []string{"O-", "O+", "A-", "A+"},
		},
		{
			recipient: "A-",
			expected:  []string{"O-", "A-"},
		},
		{
			recipient: "B+",
			expected:  []string{"O-", "O+", "B-", "B+"},
		},
		{
			recipient: "B-",
			expected:  []string{"O-", "B-"},
		},
		{
			recipient: "O+",
			expected:  []string{"O-", "O+"},
		},
		{
			recipient: "O-",
			expected:  []string{"O-"},
		},
	}

	for _, tt := range tests {
		got, err := models.GetCompatibleDonorBloodGroups(tt.recipient)
		if err != nil {
			t.Fatalf("Unexpected error for recipient '%s': %v", tt.recipient, err)
		}

		if len(got) != len(tt.expected) {
			t.Fatalf("For recipient '%s', expected %d compatible groups, got %d (%v)",
				tt.recipient, len(tt.expected), len(got), got)
		}

		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("For recipient '%s', expected [%d] to be '%s', got '%s'",
					tt.recipient, i, tt.expected[i], got[i])
			}
		}
	}

	// Invalid blood groups
	invalidInputs := []string{"", "XYZ", "C+", "123", "a+"}
	for _, inv := range invalidInputs {
		_, err := models.GetCompatibleDonorBloodGroups(inv)
		if err == nil {
			t.Errorf("Expected error for invalid blood group '%s', got nil", inv)
		}
	}
}

func TestCompatibility_IsBloodCompatible(t *testing.T) {
	// O- can donate to everyone
	allGroups := []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}
	for _, rec := range allGroups {
		if !models.IsBloodCompatible("O-", rec) {
			t.Errorf("Expected O- to be compatible with recipient '%s'", rec)
		}
	}

	// AB+ can only donate to AB+
	for _, rec := range allGroups {
		expected := rec == "AB+"
		if models.IsBloodCompatible("AB+", rec) != expected {
			t.Errorf("Expected AB+ to %s compatibility to be %v", rec, expected)
		}
	}

	// B+ can donate to B+ and AB+, but not A+ or O+
	if !models.IsBloodCompatible("B+", "B+") {
		t.Errorf("Expected B+ to be compatible with B+")
	}
	if !models.IsBloodCompatible("B+", "AB+") {
		t.Errorf("Expected B+ to be compatible with AB+")
	}
	if models.IsBloodCompatible("B+", "A+") {
		t.Errorf("Expected B+ to NOT be compatible with A+")
	}
	if models.IsBloodCompatible("B+", "O+") {
		t.Errorf("Expected B+ to NOT be compatible with O+")
	}

	// Invalid inputs should return false
	if models.IsBloodCompatible("INVALID", "B+") {
		t.Errorf("Expected invalid donor group to be false")
	}
	if models.IsBloodCompatible("B+", "INVALID") {
		t.Errorf("Expected invalid recipient group to be false")
	}
}
