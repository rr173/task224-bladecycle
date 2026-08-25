package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task224-bladecycle/internal/service"
	"task224-bladecycle/internal/store"
)

func TestBug05TelemetryAfterAcquisitionIsRejectedAndNotPersisted(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "telemetry-lifecycle.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-LIFECYCLE", "lifecycle", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("finished acquisition", "TF-LIFECYCLE", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err = app.Trials.Start(trial.ID); err != nil {
		t.Fatalf("start trial: %v", err)
	}
	if _, err = app.Trials.FinishAcquisition(trial.ID); err != nil {
		t.Fatalf("finish acquisition: %v", err)
	}

	server := httptest.NewServer(New(app).Handler())
	defer server.Close()
	payload := map[string]interface{}{
		"channel_index": 1,
		"sensor_id":     "S-LIFECYCLE",
		"seq_no":        1,
		"rpm":           1200,
		"temperature":   80,
		"strain":        []float64{0, 10, 0, 10, 0},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode telemetry: %v", err)
	}
	resp, err := http.Post(server.URL+"/api/trials/"+itoa(trial.ID)+"/telemetry", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post telemetry: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("telemetry after acquisition status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}

	resp, err = http.Get(server.URL + "/api/trials/" + itoa(trial.ID) + "/telemetry")
	if err != nil {
		t.Fatalf("list telemetry: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list telemetry status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var segments []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&segments); err != nil {
		t.Fatalf("decode telemetry list: %v", err)
	}
	if len(segments) != 0 {
		t.Fatalf("persisted telemetry after acquisition: %d segments", len(segments))
	}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	return string(buf)
}
