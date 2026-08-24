package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"task224-bladecycle/internal/service"
	"task224-bladecycle/internal/store"
)

func TestBug01CancelledTelemetryRequestDoesNotPersist(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	material, err := app.Mats.Create("MAT-CANCEL", "cancel test", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("cancelled telemetry", "TF-CANCEL", material.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	trial, err = app.Trials.Start(trial.ID)
	if err != nil {
		t.Fatalf("start trial: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/trials/"+strconv.FormatInt(trial.ID, 10)+"/telemetry", bytes.NewBufferString(`{"channel_index":0,"sensor_id":"cancel-sensor","seq_no":1,"rpm":12000,"temperature":650,"strain":[0,120,0,120,0]}`)).WithContext(ctx)
	rec := httptest.NewRecorder()
	New(app).Handler().ServeHTTP(rec, req)
	if rec.Code == http.StatusCreated {
		t.Fatalf("cancelled request unexpectedly succeeded: %s", rec.Body.String())
	}
	segments, err := app.Tele.ListByTrial(trial.ID)
	if err != nil {
		t.Fatalf("list telemetry: %v", err)
	}
	if len(segments) != 0 {
		t.Fatalf("cancelled request persisted %d segment(s)", len(segments))
	}
}
