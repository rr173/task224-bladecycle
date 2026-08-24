package service

import (
	"database/sql"

	"task224-bladecycle/internal/damage"
	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/store"
)

// SnapshotService 管理寿命快照（冻结损伤评估结论）。
type SnapshotService struct {
	snaps *store.SnapshotStore
	damage *store.DamageStore
}

func NewSnapshotService(db *sql.DB) *SnapshotService {
	return &SnapshotService{
		snaps: store.NewSnapshotStore(db),
		damage: store.NewDamageStore(db),
	}
}

// Publish 基于最新损伤记录发布一条寿命快照。
//
// 规则：快照版本递增；发布新快照会将已发布旧快照标记为 superseded；
// 冻结时记录总损伤、总循环、剩余寿命百分比与是否超阈值。
func (s *SnapshotService) Publish(trialID int64) (*model.Snapshot, error) {
	rec, err := s.damage.LatestByTrial(trialID)
	if err == sql.ErrNoRows {
		return nil, model.NewInvalidState("trial %d has no damage record to snapshot", trialID)
	}
	if err != nil {
		return nil, err
	}
	version, err := s.snaps.NextVersion(trialID)
	if err != nil {
		return nil, err
	}
	snap := &model.Snapshot{
		TrialID:           trialID,
		Version:           version,
		Status:            model.SnapshotPublished,
		TotalDamage:       rec.TotalDamage,
		TotalCycles:       rec.TotalCycles,
		RemainingLifePct:  damage.RemainingLife(rec.TotalDamage),
		ThresholdExceeded: rec.ThresholdMet,
	}
	// 旧已发布快照标记为替代。
	if err := s.snaps.SupersedePublished(trialID); err != nil {
		return nil, err
	}
	id, err := s.snaps.Insert(snap)
	if err != nil {
		return nil, err
	}
	snap.ID = id
	return snap, nil
}

// Get 按 ID 查询快照。
func (s *SnapshotService) Get(id int64) (*model.Snapshot, error) {
	snap, err := s.snaps.Get(id)
	if err == sql.ErrNoRows {
		return nil, model.NewNotFound("snapshot %d not found", id)
	}
	return snap, err
}

// ListByTrial 列出某试验全部快照。
func (s *SnapshotService) ListByTrial(trialID int64) ([]*model.Snapshot, error) {
	return s.snaps.ListByTrial(trialID)
}
