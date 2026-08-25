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
}

func NewSnapshotService(db *sql.DB) *SnapshotService {
	return &SnapshotService{
		snaps: store.NewSnapshotStore(db),
	}
}

// Publish 基于最新损伤记录发布一条寿命快照。
//
// 规则：快照版本递增；发布新快照会将已发布旧快照标记为 superseded；
// 冻结时记录总损伤、总循环、剩余寿命百分比与是否超阈值。
//
// 并发安全：读损伤、分配版本号、替代旧快照、写入新快照在 store 层的单个
// 事务内原子完成（_txlock=immediate 串行化写事务），避免同一试验短时间多次
// 发布请求交错导致的发布失败或重复版本号。
func (s *SnapshotService) Publish(trialID int64) (*model.Snapshot, error) {
	snap, err := s.snaps.Publish(trialID, func(rec *model.DamageRecord) *model.Snapshot {
		return &model.Snapshot{
			TotalDamage:       rec.TotalDamage,
			TotalCycles:       rec.TotalCycles,
			RemainingLifePct:  damage.RemainingLife(rec.TotalDamage),
			ThresholdExceeded: rec.ThresholdMet,
		}
	})
	if err == sql.ErrNoRows {
		return nil, model.NewInvalidState("trial %d has no damage record to snapshot", trialID)
	}
	return snap, err
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
