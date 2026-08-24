package service

import (
	"path/filepath"
	"sync"
	"testing"

	"task224-bladecycle/internal/store"
)

func TestBug06ConcurrentTrialStartAllowsOnlyOneTransition(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "trial-start.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-TRIAL-START", "trial start", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("concurrent start", "TF-START-RACE", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}

	const workers = 20
	start := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := app.Trials.Start(trial.ID)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent start successes = %d, want 1", successes)
	}
	final, err := app.Trials.Get(trial.ID)
	if err != nil {
		t.Fatalf("get final trial: %v", err)
	}
	if final.Status != "running" {
		t.Fatalf("final trial status = %s, want running", final.Status)
	}
}
