package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"lifelink/internal/models"
)

func TestMatchEndpoints_SuccessWithCity(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	// Donor 1: B+, Dhaka, Available (Compatible & Matching City -> Match)
	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor One",
		Email:        "d1@example.com",
		Phone:        "+8801700000001",
		BloodGroup:   "B+",
		City:         "Dhaka",
		Availability: true,
	})

	// Donor 2: O-, Dhaka, Available (Universal Donor & Matching City -> Match)
	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Two",
		Email:        "d2@example.com",
		Phone:        "+8801700000002",
		BloodGroup:   "O-",
		City:         "Dhaka",
		Availability: true,
	})

	// Donor 3: B-, Dhaka, Available (Compatible & Matching City -> Match)
	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Three",
		Email:        "d3@example.com",
		Phone:        "+8801700000003",
		BloodGroup:   "B-",
		City:         "Dhaka",
		Availability: true,
	})

	// Donor 4: A+, Dhaka, Available (Incompatible Blood Group -> Filtered out)
	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Four",
		Email:        "d4@example.com",
		Phone:        "+8801700000004",
		BloodGroup:   "A+",
		City:         "Dhaka",
		Availability: true,
	})

	// Donor 5: B+, Chittagong, Available (Different City -> Filtered out by city)
	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Five",
		Email:        "d5@example.com",
		Phone:        "+8801700000005",
		BloodGroup:   "B+",
		City:         "Chittagong",
		Availability: true,
	})

	// Donor 6: B+, Dhaka, NOT Available (Availability=false -> Filtered out)
	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Six",
		Email:        "d6@example.com",
		Phone:        "+8801700000006",
		BloodGroup:   "B+",
		City:         "Dhaka",
		Availability: false,
	})

	// GET /api/matches?bloodGroup=B+&city=Dhaka
	req, _ := http.NewRequest(http.MethodGet, "/api/matches?bloodGroup=B+&city=Dhaka", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var matches []models.Donor
	if err := json.Unmarshal(w.Body.Bytes(), &matches); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if len(matches) != 3 {
		t.Fatalf("Expected exactly 3 matching donors, got %d: %+v", len(matches), matches)
	}

	// Verify all returned donors are compatible and available in Dhaka
	for _, m := range matches {
		if !m.Availability {
			t.Errorf("Expected donor to be available, but got: %+v", m)
		}
		if m.City != "Dhaka" {
			t.Errorf("Expected city 'Dhaka', got '%s'", m.City)
		}
		if m.BloodGroup != "B+" && m.BloodGroup != "O-" && m.BloodGroup != "B-" {
			t.Errorf("Unexpected blood group '%s' for B+ recipient", m.BloodGroup)
		}
	}
}

func TestMatchEndpoints_WithoutCityFilter(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Dhaka",
		Email:        "dhaka@example.com",
		Phone:        "+8801700000001",
		BloodGroup:   "B+",
		City:         "Dhaka",
		Availability: true,
	})
	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Sylhet",
		Email:        "sylhet@example.com",
		Phone:        "+8801700000002",
		BloodGroup:   "B+",
		City:         "Sylhet",
		Availability: true,
	})

	// Query without city parameter: should return matching donors from all cities
	req, _ := http.NewRequest(http.MethodGet, "/api/matches?bloodGroup=B+", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", w.Code)
	}

	var matches []models.Donor
	if err := json.Unmarshal(w.Body.Bytes(), &matches); err != nil {
		t.Fatalf("Failed to decode matches: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("Expected 2 matches without city filter, got %d", len(matches))
	}
}

func TestMatchEndpoints_UrlEncodedPlus(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor O-",
		Email:        "o_minus@example.com",
		Phone:        "+8801700000001",
		BloodGroup:   "O-",
		City:         "Dhaka",
		Availability: true,
	})

	// Using B%2B explicitly encoded
	req, _ := http.NewRequest(http.MethodGet, "/api/matches?bloodGroup=B%2B&city=Dhaka", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK for encoded plus, got %d", w.Code)
	}

	var matches []models.Donor
	_ = json.Unmarshal(w.Body.Bytes(), &matches)
	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}
}

func TestMatchEndpoints_SnakeCaseQueryParam(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor AB+",
		Email:        "ab_plus@example.com",
		Phone:        "+8801700000001",
		BloodGroup:   "AB+",
		City:         "Dhaka",
		Availability: true,
	})

	// Using blood_group snake_case query param
	req, _ := http.NewRequest(http.MethodGet, "/api/matches?blood_group=AB+&city=Dhaka", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK for snake_case param, got %d", w.Code)
	}

	var matches []models.Donor
	_ = json.Unmarshal(w.Body.Bytes(), &matches)
	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}
}

func TestMatchEndpoints_EmptyMatches(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	// No donors in repo
	req, _ := http.NewRequest(http.MethodGet, "/api/matches?bloodGroup=O-", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", w.Code)
	}

	// Body should be an initialized empty array "[]", not "null"
	if w.Body.String() != "[]" {
		t.Errorf("Expected '[]' for no matches, got '%s'", w.Body.String())
	}
}

func TestMatchEndpoints_MissingBloodGroup(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	req, _ := http.NewRequest(http.MethodGet, "/api/matches", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request on missing bloodGroup, got %d", w.Code)
	}

	var errResp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp["error"] != "bloodGroup query parameter is required" {
		t.Errorf("Expected 'bloodGroup query parameter is required', got '%s'", errResp["error"])
	}
}

func TestMatchEndpoints_InvalidBloodGroup(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	req, _ := http.NewRequest(http.MethodGet, "/api/matches?bloodGroup=XYZ", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request on invalid bloodGroup, got %d", w.Code)
	}

	var errResp map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp["error"] != "invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-" {
		t.Errorf("Unexpected error response: %s", errResp["error"])
	}
}

func TestMatchEndpoints_CaseInsensitiveCity(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	_ = repo.Create(context.Background(), &models.Donor{
		Name:         "Donor Dhaka",
		Email:        "dhaka@example.com",
		Phone:        "+8801700000001",
		BloodGroup:   "O+",
		City:         "Dhaka",
		Availability: true,
	})

	// Query with lowercase city "dhaka"
	req, _ := http.NewRequest(http.MethodGet, "/api/matches?bloodGroup=O+&city=dhaka", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", w.Code)
	}

	var matches []models.Donor
	_ = json.Unmarshal(w.Body.Bytes(), &matches)
	if len(matches) != 1 {
		t.Errorf("Expected 1 match with case-insensitive city, got %d", len(matches))
	}
}

func TestMatchEndpoints_DatabaseError(t *testing.T) {
	repo := newMockDonorRepo()
	repo.listErr = errors.New("simulated database failure")
	router := setupTestRouter(repo)

	req, _ := http.NewRequest(http.MethodGet, "/api/matches?bloodGroup=A+", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 Internal Server Error, got %d", w.Code)
	}
}
