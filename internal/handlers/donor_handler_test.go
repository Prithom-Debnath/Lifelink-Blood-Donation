package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
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

// mockDonorRepo is an in-memory mock implementation of repositories.DonorRepository for fast handler tests.
type mockDonorRepo struct {
	mu     sync.RWMutex
	donors map[string]models.Donor
}

func newMockDonorRepo() *mockDonorRepo {
	return &mockDonorRepo{
		donors: make(map[string]models.Donor),
	}
}

func (m *mockDonorRepo) Create(ctx context.Context, donor *models.Donor) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if donor.ID.IsZero() {
		donor.ID = bson.NewObjectID()
	}
	now := time.Now().UTC()
	donor.CreatedAt = now
	donor.UpdatedAt = now
	m.donors[donor.ID.Hex()] = *donor
	return nil
}

func (m *mockDonorRepo) List(ctx context.Context) ([]models.Donor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]models.Donor, 0, len(m.donors))
	for _, d := range m.donors {
		list = append(list, d)
	}
	return list, nil
}

func (m *mockDonorRepo) FindByID(ctx context.Context, id string) (*models.Donor, error) {
	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return nil, repositories.ErrInvalidID
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	donor, exists := m.donors[id]
	if !exists {
		return nil, repositories.ErrNotFound
	}
	return &donor, nil
}

func (m *mockDonorRepo) Update(ctx context.Context, id string, donor *models.Donor) error {
	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return repositories.ErrInvalidID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.donors[id]
	if !exists {
		return repositories.ErrNotFound
	}

	donor.ID = existing.ID
	donor.CreatedAt = existing.CreatedAt
	donor.UpdatedAt = time.Now().UTC()
	m.donors[id] = *donor
	return nil
}

func (m *mockDonorRepo) Delete(ctx context.Context, id string) error {
	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return repositories.ErrInvalidID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.donors[id]; !exists {
		return repositories.ErrNotFound
	}
	delete(m.donors, id)
	return nil
}

func setupTestRouter(repo repositories.DonorRepository) http.Handler {
	cfg := &config.Config{
		Port:   "8080",
		AppEnv: "test",
	}
	return routes.SetupRouter(cfg, repo, nil)
}

func TestDonorEndpoints_CRUD(t *testing.T) {
	repo := newMockDonorRepo()
	router := setupTestRouter(repo)

	// 1. Test POST /api/donors (Success: 201 Created)
	payload := []byte(`{
		"name": "Nusrat Jahan",
		"email": "nusrat@example.com",
		"phone": "+8801819000000",
		"blood_group": "AB+",
		"city": "Dhaka",
		"availability": true
	}`)

	req, _ := http.NewRequest(http.MethodPost, "/api/donors", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 on POST, got %d. Body: %s", w.Code, w.Body.String())
	}

	var created models.Donor
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to parse created donor JSON: %v", err)
	}
	if created.ID.IsZero() || created.Name != "Nusrat Jahan" {
		t.Errorf("Created donor data mismatch: %+v", created)
	}

	donorID := created.ID.Hex()

	// 2. Test POST /api/donors (Bad Request: 400 on invalid JSON)
	badReq, _ := http.NewRequest(http.MethodPost, "/api/donors", bytes.NewBufferString("{invalid-json"))
	badReq.Header.Set("Content-Type", "application/json")
	wBad := httptest.NewRecorder()
	router.ServeHTTP(wBad, badReq)
	if wBad.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 on malformed JSON, got %d", wBad.Code)
	}

	// 3. Test GET /api/donors (Success: 200 OK)
	reqList, _ := http.NewRequest(http.MethodGet, "/api/donors", nil)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/donors, got %d", wList.Code)
	}

	var donors []models.Donor
	if err := json.Unmarshal(wList.Body.Bytes(), &donors); err != nil {
		t.Fatalf("Failed to decode donors list: %v", err)
	}
	if len(donors) != 1 {
		t.Errorf("Expected 1 donor in list, got %d", len(donors))
	}

	// 4. Test GET /api/donors/:id (Success: 200 OK)
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/donors/"+donorID, nil)
	wGet := httptest.NewRecorder()
	router.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on GET /api/donors/:id, got %d", wGet.Code)
	}

	var fetched models.Donor
	json.Unmarshal(wGet.Body.Bytes(), &fetched)
	if fetched.Email != "nusrat@example.com" {
		t.Errorf("Expected email nusrat@example.com, got %s", fetched.Email)
	}

	// 5. Test GET /api/donors/:id (Bad Request: 400 on invalid ObjectID format)
	reqInvalidID, _ := http.NewRequest(http.MethodGet, "/api/donors/invalid-hex-id", nil)
	wInvalidID := httptest.NewRecorder()
	router.ServeHTTP(wInvalidID, reqInvalidID)
	if wInvalidID.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 on invalid ID, got %d", wInvalidID.Code)
	}

	// 6. Test GET /api/donors/:id (Not Found: 404 on missing record)
	nonExistentHex := bson.NewObjectID().Hex()
	reqNotFound, _ := http.NewRequest(http.MethodGet, "/api/donors/"+nonExistentHex, nil)
	wNotFound := httptest.NewRecorder()
	router.ServeHTTP(wNotFound, reqNotFound)
	if wNotFound.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 on missing donor, got %d", wNotFound.Code)
	}

	// 7. Test PUT /api/donors/:id (Success: 200 OK)
	updatePayload := []byte(`{
		"name": "Nusrat Jahan Chowdhury",
		"email": "nusrat.c@example.com",
		"phone": "+8801819000000",
		"blood_group": "AB+",
		"city": "Sylhet",
		"availability": false
	}`)
	reqPut, _ := http.NewRequest(http.MethodPut, "/api/donors/"+donorID, bytes.NewBuffer(updatePayload))
	reqPut.Header.Set("Content-Type", "application/json")
	wPut := httptest.NewRecorder()
	router.ServeHTTP(wPut, reqPut)

	if wPut.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on PUT, got %d. Body: %s", wPut.Code, wPut.Body.String())
	}

	var updated models.Donor
	json.Unmarshal(wPut.Body.Bytes(), &updated)
	if updated.City != "Sylhet" || updated.Availability != false {
		t.Errorf("Update data mismatch: %+v", updated)
	}

	// 8. Test PUT /api/donors/:id (Not Found: 404 on missing record)
	reqPutNotFound, _ := http.NewRequest(http.MethodPut, "/api/donors/"+nonExistentHex, bytes.NewBuffer(updatePayload))
	reqPutNotFound.Header.Set("Content-Type", "application/json")
	wPutNotFound := httptest.NewRecorder()
	router.ServeHTTP(wPutNotFound, reqPutNotFound)
	if wPutNotFound.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 on PUT missing donor, got %d", wPutNotFound.Code)
	}

	// 9. Test DELETE /api/donors/:id (Success: 200 OK)
	reqDelete, _ := http.NewRequest(http.MethodDelete, "/api/donors/"+donorID, nil)
	wDelete := httptest.NewRecorder()
	router.ServeHTTP(wDelete, reqDelete)

	if wDelete.Code != http.StatusOK {
		t.Fatalf("Expected status 200 on DELETE, got %d. Body: %s", wDelete.Code, wDelete.Body.String())
	}

	// 10. Test DELETE /api/donors/:id (Not Found: 404 after deletion)
	reqDeleteAgain, _ := http.NewRequest(http.MethodDelete, "/api/donors/"+donorID, nil)
	wDeleteAgain := httptest.NewRecorder()
	router.ServeHTTP(wDeleteAgain, reqDeleteAgain)

	if wDeleteAgain.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 on deleting already deleted donor, got %d", wDeleteAgain.Code)
	}
}
