package service

import (
	"path/filepath"
	"testing"

	"task224-bladecycle/internal/store"
)

func TestBug09TrialCreationRequiresExistingMaterial(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "material-reference.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	if _, err = app.Trials.Create("orphan trial", "TF-ORPHAN", 99999, 1000); err == nil {
		t.Fatal("trial with missing material unexpectedly succeeded")
	}
	trials, err := app.Trials.List()
	if err != nil {
		t.Fatalf("list trials: %v", err)
	}
	if len(trials) != 0 {
		t.Fatalf("orphan trial was persisted: %d records", len(trials))
	}
}
