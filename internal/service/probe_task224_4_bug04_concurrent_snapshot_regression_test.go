package service

import (
	"path/filepath"
	"sync"
	"testing"

	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/store"
)

func TestBug04ConcurrentSnapshotPublicationUsesUniqueVersions(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "snapshots.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-CONCURRENT-SNAPSHOT", "snapshot race", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("concurrent snapshots", "TF-SNAPSHOT-RACE", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err = app.Trials.Start(trial.ID); err != nil {
		t.Fatalf("start trial: %v", err)
	}
	if _, err = app.Trials.FinishAcquisition(trial.ID); err != nil {
		t.Fatalf("finish acquisition: %v", err)
	}
	if _, err = store.NewDamageStore(db).Insert(&model.DamageRecord{TrialID: trial.ID, MaterialBatchID: mat.ID, TotalCycles: 10, TotalDamage: 0.2, MaxAmplitude: 20}); err != nil {
		t.Fatalf("insert damage: %v", err)
	}

	const workers = 20
	start := make(chan struct{})
	versions := make(chan int, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			snap, err := app.Snaps.Publish(trial.ID)
			if err != nil {
				errs <- err
				return
			}
			versions <- snap.Version
		}()
	}
	close(start)
	wg.Wait()
	close(versions)
	close(errs)
	for err := range errs {
		t.Errorf("concurrent publish error: %v", err)
	}
	seen := map[int]bool{}
	for version := range versions {
		if seen[version] {
			t.Errorf("duplicate snapshot version %d", version)
		}
		seen[version] = true
	}
	if len(seen) != workers {
		t.Fatalf("published versions = %d, want %d", len(seen), workers)
	}
}
