package service

import (
	"path/filepath"
	"testing"

	"task224-bladecycle/internal/store"
)

func TestBug07AnalysisFailurePreservesPreviousResults(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "analysis.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-ANALYSIS-ATOMIC", "analysis atomic", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("analysis atomic", "TF-ANALYSIS-ATOMIC", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err = app.Trials.Start(trial.ID); err != nil {
		t.Fatalf("start trial: %v", err)
	}
	if _, err = app.Tele.Ingest(trial, 1, "S-ANALYSIS", 1, 1200, 80, []float64{0, 10, 0, 10, 0}); err != nil {
		t.Fatalf("ingest telemetry: %v", err)
	}
	trial, err = app.Trials.FinishAcquisition(trial.ID)
	if err != nil {
		t.Fatalf("finish acquisition: %v", err)
	}
	if _, err = app.Cycles.Analyze(trial); err != nil {
		t.Fatalf("initial analysis: %v", err)
	}
	oldCycles, err := app.Cycles.ListCycles(trial.ID)
	if err != nil {
		t.Fatalf("list initial cycles: %v", err)
	}
	oldDamage, err := app.Cycles.DamageHistory(trial.ID)
	if err != nil {
		t.Fatalf("list initial damage: %v", err)
	}
	if len(oldCycles) == 0 || len(oldDamage) == 0 {
		t.Fatalf("initial analysis did not persist results: cycles=%d damage=%d", len(oldCycles), len(oldDamage))
	}
	if _, err = db.Exec(`CREATE TRIGGER fail_cycle_insert BEFORE INSERT ON cycles BEGIN SELECT RAISE(ABORT, 'forced cycle insert failure'); END`); err != nil {
		t.Fatalf("install failure trigger: %v", err)
	}
	if _, err = app.Cycles.Analyze(trial); err == nil {
		t.Fatal("failed re-analysis unexpectedly succeeded")
	}

	cycles, err := app.Cycles.ListCycles(trial.ID)
	if err != nil {
		t.Fatalf("list cycles after failed analysis: %v", err)
	}
	damage, err := app.Cycles.DamageHistory(trial.ID)
	if err != nil {
		t.Fatalf("list damage after failed analysis: %v", err)
	}
	if len(cycles) != len(oldCycles) || len(damage) != len(oldDamage) {
		t.Fatalf("failed analysis discarded previous results: cycles %d/%d damage %d/%d", len(cycles), len(oldCycles), len(damage), len(oldDamage))
	}
}
