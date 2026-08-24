package store

import (
	"path/filepath"
	"testing"

	"task224-bladecycle/internal/model"
)

func TestTelemetryPersistenceAndIdempotencyAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "bladecycle.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	trialStore := NewTrialStore(db)
	trialID, err := trialStore.Insert("persistence trial", "TF-2000", 7, 1000)
	if err != nil {
		db.Close()
		t.Fatalf("insert trial: %v", err)
	}

	telemetryStore := NewTelemetryStore(db)
	segment := &model.TelemetrySegment{
		TrialID:      trialID,
		ChannelIndex: 2,
		SeqNo:        11,
		RPM:          12000,
		Temperature:  650,
		StrainPeaks:  []float64{0, 120, 0},
		Status:       model.SegmentValid,
	}
	segmentID, err := telemetryStore.InsertSegment(segment)
	if err != nil {
		db.Close()
		t.Fatalf("insert telemetry segment: %v", err)
	}
	if _, err := telemetryStore.InsertSegment(segment); err != model.ErrDuplicate {
		db.Close()
		t.Fatalf("duplicate insert error = %v, want %v", err, model.ErrDuplicate)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close first database: %v", err)
	}

	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("reopen database: %v", err)
	}
	defer db.Close()

	reloadedTrial, err := NewTrialStore(db).Get(trialID)
	if err != nil {
		t.Fatalf("reload trial: %v", err)
	}
	if reloadedTrial.Name != "persistence trial" || reloadedTrial.Status != model.TrialReady {
		t.Fatalf("reloaded trial = %#v", reloadedTrial)
	}
	reloadedSegment, err := NewTelemetryStore(db).GetSegment(segmentID)
	if err != nil {
		t.Fatalf("reload telemetry segment: %v", err)
	}
	if reloadedSegment.SeqNo != segment.SeqNo || reloadedSegment.Status != model.SegmentValid {
		t.Fatalf("reloaded segment = %#v", reloadedSegment)
	}
	if len(reloadedSegment.StrainPeaks) != 3 || reloadedSegment.StrainPeaks[1] != 120 {
		t.Fatalf("reloaded peaks = %v", reloadedSegment.StrainPeaks)
	}
}
