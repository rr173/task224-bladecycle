package service

import (
	"path/filepath"
	"testing"

	"task224-bladecycle/internal/store"
)

func TestBug08AnalysisWithoutValidTelemetryIsRejected(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "empty-analysis.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-EMPTY-ANALYSIS", "empty analysis", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("empty analysis", "TF-EMPTY-ANALYSIS", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err = app.Trials.Start(trial.ID); err != nil {
		t.Fatalf("start trial: %v", err)
	}
	trial, err = app.Trials.FinishAcquisition(trial.ID)
	if err != nil {
		t.Fatalf("finish acquisition: %v", err)
	}
	if _, err = app.Cycles.Analyze(trial); err == nil {
		t.Fatal("analysis without telemetry unexpectedly succeeded")
	}
	records, err := app.Cycles.DamageHistory(trial.ID)
	if err != nil {
		t.Fatalf("list damage history: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("empty analysis created %d damage records", len(records))
	}
}
