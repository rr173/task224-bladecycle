package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"task224-bladecycle/internal/service"
	"task224-bladecycle/internal/store"
)

func TestBug02OutOfOrderTelemetrySequenceIsRejected(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	mat, err := app.Mats.Create("MAT-ORDER", "order test", 200, -0.12, 300, "goodman")
	if err != nil {
		t.Fatalf("create material: %v", err)
	}
	trial, err := app.Trials.Create("sequence order", "TF-ORDER", mat.ID, 1000)
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if _, err = app.Trials.Start(trial.ID); err != nil {
		t.Fatalf("start trial: %v", err)
	}
	h := httptest.NewServer(New(app).Handler())
	defer h.Close()
	post := func(seq int) *http.Response {
		body := map[string]interface{}{"channel_index": 0, "sensor_id": "order-sensor", "seq_no": seq, "rpm": 12000, "temperature": 650, "strain": []float64{0, 120, 0, 120, 0}}
		payload, _ := json.Marshal(body)
		resp, err := http.Post(h.URL+"/api/trials/"+strconv.FormatInt(trial.ID, 10)+"/telemetry", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("post seq %d: %v", seq, err)
		}
		return resp
	}
	first := post(5)
	if first.StatusCode != http.StatusCreated {
		first.Body.Close()
		t.Fatalf("first status = %d", first.StatusCode)
	}
	first.Body.Close()
	second := post(3)
	defer second.Body.Close()
	if second.StatusCode != http.StatusConflict {
		t.Fatalf("out-of-order status = %d, want %d", second.StatusCode, http.StatusConflict)
	}
	segments, err := app.Tele.ListByTrial(trial.ID)
	if err != nil {
		t.Fatalf("list telemetry: %v", err)
	}
	if len(segments) != 1 || segments[0].SeqNo != 5 {
		t.Fatalf("stored segments = %#v", segments)
	}
}
