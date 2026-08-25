package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/service"
	"task224-bladecycle/internal/store"
)

func TestBug03SnapshotRequiresCompletedAnalysis(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-SNAPSHOT", "snapshot test", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("premature snapshot", "TF-SNAPSHOT", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	_, err = store.NewDamageStore(db).Insert(&model.DamageRecord{TrialID: trial.ID, MaterialBatchID: mat.ID, TotalCycles: 3, TotalDamage: 0.2, MaxAmplitude: 10})
	if err != nil {
		t.Fatalf("insert damage: %v", err)
	}
	h := httptest.NewServer(New(app).Handler())
	defer h.Close()
	resp, err := http.Post(h.URL+"/api/trials/"+strconv.FormatInt(trial.ID, 10)+"/snapshots", "application/json", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("publish snapshot: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("premature snapshot status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
	snaps, err := app.Snaps.ListByTrial(trial.ID)
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snaps) != 0 {
		t.Fatalf("premature snapshot persisted: %#v", snaps)
	}
}
