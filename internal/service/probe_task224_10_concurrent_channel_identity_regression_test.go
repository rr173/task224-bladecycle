package service

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/store"
)

func TestBug10ConcurrentChannelRegistrationPreservesSensorIdentity(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "channel-identity.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-CHANNEL-RACE", "channel race", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("channel race", "TF-CHANNEL-RACE", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err = app.Trials.Start(trial.ID); err != nil {
		t.Fatalf("start trial: %v", err)
	}

	const workers = 20
	start := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := app.Tele.Ingest(trial, 1, fmt.Sprintf("SENSOR-%02d", i), int64(i+1), 1200, 80, []float64{0, 10, 0, 10, 0})
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if model.IsConflict(err) {
			conflicts++
		} else {
			t.Errorf("unexpected concurrent channel error: %v", err)
		}
	}
	if successes != 1 || conflicts != workers-1 {
		t.Fatalf("channel registration results = successes %d conflicts %d, want 1/%d", successes, conflicts, workers-1)
	}
	segments, err := app.Tele.ListByTrial(trial.ID)
	if err != nil {
		t.Fatalf("list telemetry: %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("persisted segments = %d, want 1", len(segments))
	}
}
