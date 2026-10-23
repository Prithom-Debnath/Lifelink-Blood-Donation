package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"lifelink/internal/config"
	"lifelink/internal/models"
	"lifelink/internal/repositories"
	"lifelink/internal/routes"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// mockBloodRequestRepo is an in-memory mock implementation of repositories.BloodRequestRepository for handler tests.
type mockBloodRequestRepo struct {
	mu          sync.RWMutex
	requests    map[string]models.BloodRequest
	createErr   error
	listErr     error
	findByIDErr error
	updateErr   error
}

func newMockBloodRequestRepo() *mockBloodRequestRepo {
	return &mockBloodRequestRepo{
		requests: make(map[string]models.BloodRequest),
	}
}

func (m *mockBloodRequestRepo) Create(ctx context.Context, req *models.BloodRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createErr != nil {
		return m.createErr
	}

	if req.ID.IsZero() {
		req.ID = bson.NewObjectID()
	}
	if req.Status == "" {
		req.Status = models.StatusOpen
	}
	now := time.Now().UTC()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	req.UpdatedAt = now

	m.requests[req.ID.Hex()] = *req
	return nil
}

func (m *mockBloodRequestRepo) List(ctx context.Context) ([]models.BloodRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.listErr != nil {
		return nil, m.listErr
	}

	list := make([]models.BloodRequest, 0, len(m.requests))
	for _, r := range m.requests {
		list = append(list, r)
	}
	return list, nil
}

func (m *mockBloodRequestRepo) FindByID(ctx context.Context, id string) (*models.BloodRequest, error) {
	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return nil, repositories.ErrInvalidBloodRequestID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}

	req, exists := m.requests[id]
	if !exists {
		return nil, repositories.ErrBloodRequestNotFound
	}
	return &req, nil
}

func (m *mockBloodRequestRepo) Update(ctx context.Context, id string, req *models.BloodRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	current, exists := m.requests[id]
	if !exists {
		return repositories.ErrBloodRequestNotFound
	}
	if current.Status != models.StatusOpen {
		return repositories.ErrBloodRequestNotOpen
	}
	req.ID = current.ID
	req.CreatedAt = current.CreatedAt
	req.UpdatedAt = time.Now().UTC()
	req.Status = current.Status
	m.requests[id] = *req
	return nil
}

func (m *mockBloodRequestRepo) UpdateStatus(ctx context.Context, id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, exists := m.requests[id]
	if !exists {
		return repositories.ErrBloodRequestNotFound
	}
	if current.Status != models.StatusOpen {
		return repositories.ErrBloodRequestNotOpen
	}
	current.Status = status
	current.UpdatedAt = time.Now().UTC()
	m.requests[id] = current
	return nil
}

func setupTestRouterWithRequestRepo(repo repositories.BloodRequestRepository) http.Handler {
	cfg := &config.Config{
		Port:   "8080",
		AppEnv: "test",
	}
	return routes.SetupRouter(cfg, nil, repo, nil)
}

func TestBloodRequestEndpoints_CreateSuccess(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	payload := []byte(`{
		"patient_name": "Rahim Uddin",
		"blood_group": "B+",
		"hospital": "Dhaka Medical College Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801711223344"
	}`)

	req, _ := http.NewRequest(http.MethodPost, "/api/requests", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var created models.BloodRequest
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to parse created blood request JSON: %v", err)
	}

	if created.ID.IsZero() {
		t.Errorf("Expected non-zero ObjectID, got zero")
	}
	if created.PatientName != "Rahim Uddin" {
		t.Errorf("Expected patient_name 'Rahim Uddin', got '%s'", created.PatientName)
	}
	if created.BloodGroup != "B+" {
		t.Errorf("Expected blood_group 'B+', got '%s'", created.BloodGroup)
	}
	if created.Hospital != "Dhaka Medical College Hospital" {
		t.Errorf("Expected hospital 'Dhaka Medical College Hospital', got '%s'", created.Hospital)
	}
	if created.City != "Dhaka" {
		t.Errorf("Expected city 'Dhaka', got '%s'", created.City)
	}
	if created.ContactPhone != "+8801711223344" {
		t.Errorf("Expected contact_phone '+8801711223344', got '%s'", created.ContactPhone)
	}
	if created.Status != models.StatusOpen {
		t.Errorf("Expected status '%s', got '%s'", models.StatusOpen, created.Status)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Errorf("Expected non-zero timestamps, got CreatedAt: %v, UpdatedAt: %v", created.CreatedAt, created.UpdatedAt)
	}
}

func TestBloodRequestEndpoints_CreateCamelCase(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	payload := []byte(`{
		"patientName": "Farhana Ahmed",
		"bloodGroup": "O-",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contactPhone": "+8801811223344"
	}`)

	req, _ := http.NewRequest(http.MethodPost, "/api/requests", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created for camelCase, got %d. Body: %s", w.Code, w.Body.String())
	}

	var created models.BloodRequest
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to parse created blood request JSON: %v", err)
	}

	if created.PatientName != "Farhana Ahmed" || created.BloodGroup != "O-" || created.ContactPhone != "+8801811223344" {
		t.Errorf("Mismatch in camelCase deserialization: %+v", created)
	}
}

func TestBloodRequestEndpoints_CreateInvalidJSON(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	req, _ := http.NewRequest(http.MethodPost, "/api/requests", bytes.NewBufferString("{invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request on malformed JSON, got %d", w.Code)
	}

	var errResp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil || errResp["error"] == "" {
		t.Errorf("Expected JSON error response, got %s", w.Body.String())
	}
}

func TestBloodRequestEndpoints_CreateDatabaseError(t *testing.T) {
	repo := newMockBloodRequestRepo()
	repo.createErr = errors.New("simulated mongo write failure")
	router := setupTestRouterWithRequestRepo(repo)

	payload := []byte(`{
		"patient_name": "Test Patient",
		"blood_group": "A+",
		"hospital": "City Hospital",
		"city": "Chittagong",
		"contact_phone": "+8801911223344"
	}`)

	req, _ := http.NewRequest(http.MethodPost, "/api/requests", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 Internal Server Error, got %d", w.Code)
	}
}

func TestBloodRequestEndpoints_ListEmpty(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	req, _ := http.NewRequest(http.MethodGet, "/api/requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", w.Code)
	}

	// Body should be an empty JSON array "[]", not "null"
	if w.Body.String() != "[]" {
		t.Errorf("Expected '[]' for empty list, got '%s'", w.Body.String())
	}
}

func TestBloodRequestEndpoints_ListPopulated(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	_ = repo.Create(context.Background(), &models.BloodRequest{
		PatientName:  "Patient One",
		BloodGroup:   "A+",
		Hospital:     "Hospital A",
		City:         "Dhaka",
		ContactPhone: "+8801700000001",
	})
	_ = repo.Create(context.Background(), &models.BloodRequest{
		PatientName:  "Patient Two",
		BloodGroup:   "O+",
		Hospital:     "Hospital B",
		City:         "Sylhet",
		ContactPhone: "+8801700000002",
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", w.Code)
	}

	var requests []models.BloodRequest
	if err := json.Unmarshal(w.Body.Bytes(), &requests); err != nil {
		t.Fatalf("Failed to decode requests: %v", err)
	}

	if len(requests) != 2 {
		t.Errorf("Expected 2 requests, got %d", len(requests))
	}
}

func TestBloodRequestEndpoints_ListDatabaseError(t *testing.T) {
	repo := newMockBloodRequestRepo()
	repo.listErr = errors.New("simulated query error")
	router := setupTestRouterWithRequestRepo(repo)

	req, _ := http.NewRequest(http.MethodGet, "/api/requests", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 Internal Server Error, got %d", w.Code)
	}
}

func TestBloodRequestEndpoints_GetByIDSuccess(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	bloodReq := &models.BloodRequest{
		PatientName:  "Tariqul Islam",
		BloodGroup:   "AB+",
		Hospital:     "Apollo Hospital",
		City:         "Dhaka",
		ContactPhone: "+8801755667788",
	}
	_ = repo.Create(context.Background(), bloodReq)
	idHex := bloodReq.ID.Hex()

	req, _ := http.NewRequest(http.MethodGet, "/api/requests/"+idHex, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var fetched models.BloodRequest
	if err := json.Unmarshal(w.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("Failed to parse fetched JSON: %v", err)
	}

	if fetched.PatientName != "Tariqul Islam" || fetched.Hospital != "Apollo Hospital" {
		t.Errorf("Fetched request data mismatch: %+v", fetched)
	}
}

func TestBloodRequestEndpoints_GetByIDInvalidHex(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	req, _ := http.NewRequest(http.MethodGet, "/api/requests/not-a-valid-hex-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request on invalid ID format, got %d", w.Code)
	}

	var errResp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil || errResp["error"] != "Invalid blood request ID format" {
		t.Errorf("Unexpected error response: %s", w.Body.String())
	}
}

func TestBloodRequestEndpoints_GetByIDNotFound(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	nonExistentHex := bson.NewObjectID().Hex()
	req, _ := http.NewRequest(http.MethodGet, "/api/requests/"+nonExistentHex, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found, got %d", w.Code)
	}

	var errResp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil || errResp["error"] != "Blood request not found" {
		t.Errorf("Unexpected error response: %s", w.Body.String())
	}
}

func TestBloodRequestEndpoints_GetByIDDatabaseError(t *testing.T) {
	repo := newMockBloodRequestRepo()
	repo.findByIDErr = errors.New("simulated mongo read timeout")
	router := setupTestRouterWithRequestRepo(repo)

	validHex := bson.NewObjectID().Hex()
	req, _ := http.NewRequest(http.MethodGet, "/api/requests/"+validHex, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 Internal Server Error, got %d", w.Code)
	}
}

func TestBloodRequestEndpoints_Validation(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	testValidation := func(t *testing.T, payload string, expectedError string) {
		req, _ := http.NewRequest(http.MethodPost, "/api/requests", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected status 400 Bad Request, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse JSON response: %v", err)
		}
		if resp["error"] != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, resp["error"])
		}
	}

	// 1. Missing patient_name
	testValidation(t, `{
		"patient_name": "",
		"blood_group": "A+",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801700112233"
	}`, "patient_name is required")

	// 2. Missing blood_group
	testValidation(t, `{
		"patient_name": "Rahim",
		"blood_group": "   ",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801700112233"
	}`, "blood_group is required")

	// 3. Unsupported blood_group
	testValidation(t, `{
		"patient_name": "Rahim",
		"blood_group": "C+",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801700112233"
	}`, "invalid blood group: must be one of A+, A-, B+, B-, AB+, AB-, O+, O-")

	// 4. Missing hospital
	testValidation(t, `{
		"patient_name": "Rahim",
		"blood_group": "O+",
		"hospital": "",
		"city": "Dhaka",
		"contact_phone": "+8801700112233"
	}`, "hospital is required")

	// 5. Missing city
	testValidation(t, `{
		"patient_name": "Rahim",
		"blood_group": "O+",
		"hospital": "Square Hospital",
		"city": "   ",
		"contact_phone": "+8801700112233"
	}`, "city is required")

	// 6. Missing contact_phone
	testValidation(t, `{
		"patient_name": "Rahim",
		"blood_group": "O+",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contact_phone": ""
	}`, "contact_phone is required")

	// 7. Invalid status
	testValidation(t, `{
		"patient_name": "Rahim",
		"blood_group": "O+",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801700112233",
		"status": "pending"
	}`, "invalid status: must be one of open or fulfilled")

	// 8. Valid status should succeed (201 Created)
	validReq, _ := http.NewRequest(http.MethodPost, "/api/requests", bytes.NewBufferString(`{
		"patient_name": "Rahim",
		"blood_group": "O+",
		"hospital": "Square Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801700112233",
		"status": "open"
	}`))
	validReq.Header.Set("Content-Type", "application/json")
	wValid := httptest.NewRecorder()
	router.ServeHTTP(wValid, validReq)

	if wValid.Code != http.StatusCreated {
		t.Errorf("Expected status 201 Created on valid status, got %d. Body: %s", wValid.Code, wValid.Body.String())
	}
}

func TestBloodRequestEndpoints_UrgencyAndCoordinates(t *testing.T) {
	repo := newMockBloodRequestRepo()
	router := setupTestRouterWithRequestRepo(repo)

	// 1. Create request with critical urgency and coordinates
	payload := `{
		"patient_name": "Emergency Patient",
		"blood_group": "AB-",
		"hospital": "Evercare Hospital",
		"city": "Dhaka",
		"contact_phone": "+8801799887766",
		"urgency": "critical",
		"latitude": 23.8103,
		"longitude": 90.4125
	}`
	req, _ := http.NewRequest(http.MethodPost, "/api/requests", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var created models.BloodRequest
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	if created.Urgency != "critical" {
		t.Errorf("Expected urgency 'critical', got '%s'", created.Urgency)
	}
	if created.Latitude == nil || *created.Latitude != 23.8103 {
		t.Errorf("Expected latitude 23.8103, got %v", created.Latitude)
	}
	if created.Longitude == nil || *created.Longitude != 90.4125 {
		t.Errorf("Expected longitude 90.4125, got %v", created.Longitude)
	}

	// 2. Create requests with different urgencies for filter & sort tests
	_ = repo.Create(context.Background(), &models.BloodRequest{
		PatientName:  "Patient Normal",
		BloodGroup:   "O+",
		Hospital:     "Hospital A",
		City:         "Dhaka",
		ContactPhone: "+8801700000001",
		Urgency:      "normal",
	})
	_ = repo.Create(context.Background(), &models.BloodRequest{
		PatientName:  "Patient Urgent",
		BloodGroup:   "O+",
		Hospital:     "Hospital B",
		City:         "Dhaka",
		ContactPhone: "+8801700000002",
		Urgency:      "urgent",
	})

	// 3. Test filter by urgency: GET /api/requests?urgency=critical
	{
		filterReq, _ := http.NewRequest(http.MethodGet, "/api/requests?urgency=critical", nil)
		filterW := httptest.NewRecorder()
		router.ServeHTTP(filterW, filterReq)

		if filterW.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on urgency filter, got %d", filterW.Code)
		}

		var list []models.BloodRequest
		_ = json.Unmarshal(filterW.Body.Bytes(), &list)
		if len(list) != 1 {
			t.Fatalf("Expected 1 critical request, got %d", len(list))
		}
		if list[0].PatientName != "Emergency Patient" {
			t.Errorf("Expected 'Emergency Patient', got '%s'", list[0].PatientName)
		}
	}

	// 4. Test sort by urgency: GET /api/requests?sort=urgency (Critical > Urgent > Normal)
	{
		sortReq, _ := http.NewRequest(http.MethodGet, "/api/requests?sort=urgency", nil)
		sortW := httptest.NewRecorder()
		router.ServeHTTP(sortW, sortReq)

		if sortW.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on urgency sort, got %d", sortW.Code)
		}

		var list []models.BloodRequest
		_ = json.Unmarshal(sortW.Body.Bytes(), &list)
		if len(list) != 3 {
			t.Fatalf("Expected 3 requests, got %d", len(list))
		}
		if list[0].Urgency != "critical" {
			t.Errorf("Expected first request to be critical, got '%s'", list[0].Urgency)
		}
		if list[1].Urgency != "urgent" {
			t.Errorf("Expected second request to be urgent, got '%s'", list[1].Urgency)
		}
		if list[2].Urgency != "normal" {
			t.Errorf("Expected third request to be normal, got '%s'", list[2].Urgency)
		}
	}
}
