package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task224-bladecycle/internal/service"
	"task224-bladecycle/internal/store"
)

func TestMaterialAndTrialAPIsUseRealMuxAndPersistWrites(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	server := httptest.NewServer(New(app).Handler())
	defer server.Close()

	materialBody := bytes.NewBufferString(`{"code":"MAT-HTTP","name":"HTTP batch","fatigue_strength_coef":200,"fatigue_exponent":-0.12,"ultimate_strength":300,"mean_stress_method":"goodman"}`)
	resp, err := http.Post(server.URL+"/api/materials", "application/json", materialBody)
	if err != nil {
		t.Fatalf("create material request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create material status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	var material struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&material); err != nil {
		resp.Body.Close()
		t.Fatalf("decode material response: %v", err)
	}
	resp.Body.Close()
	if material.ID == 0 {
		t.Fatal("created material has no ID")
	}

	trialPayload := map[string]interface{}{
		"name":              "HTTP trial",
		"engine_model":      "TF-HTTP",
		"material_batch_id": material.ID,
		"sample_rate_hz":    1000,
	}
	trialJSON, err := json.Marshal(trialPayload)
	if err != nil {
		t.Fatalf("encode trial request: %v", err)
	}
	resp, err = http.Post(server.URL+"/api/trials", "application/json", bytes.NewReader(trialJSON))
	if err != nil {
		t.Fatalf("create trial request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		resp.Body.Close()
		t.Fatalf("create trial status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	var trial struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&trial); err != nil {
		resp.Body.Close()
		t.Fatalf("decode trial response: %v", err)
	}
	resp.Body.Close()
	if trial.ID == 0 || trial.Status != "ready" {
		t.Fatalf("created trial = %#v", trial)
	}

	resp, err = http.Get(server.URL + "/api/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
