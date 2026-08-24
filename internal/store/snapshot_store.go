package store

import (
	"database/sql"

	"task224-bladecycle/internal/model"
)

// SnapshotStore 管理寿命快照（冻结损伤评估结论）的持久化。
type SnapshotStore struct {
	db *sql.DB
}

func NewSnapshotStore(db *sql.DB) *SnapshotStore { return &SnapshotStore{db: db} }

// NextVersion 返回某试验下一个快照版本号（从 1 开始递增）。
func (s *SnapshotStore) NextVersion(trialID int64) (int, error) {
	var v sql.NullInt64
	err := s.db.QueryRow(
		`SELECT MAX(version) FROM snapshots WHERE trial_id = ?`, trialID,
	).Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 1, nil
	}
	return int(v.Int64) + 1, nil
}

// Insert 写入一条寿命快照。UNIQUE(trial_id, version) 保证版本唯一。
func (s *SnapshotStore) Insert(snap *model.Snapshot) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO snapshots (trial_id, version, status, total_damage, total_cycles, remaining_life_pct, threshold_exceeded, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.TrialID, snap.Version, string(snap.Status), snap.TotalDamage, snap.TotalCycles,
		snap.RemainingLifePct, boolToInt(snap.ThresholdExceeded), nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Get 按 ID 查询快照。
func (s *SnapshotStore) Get(id int64) (*model.Snapshot, error) {
	row := s.db.QueryRow(
		`SELECT id, trial_id, version, status, total_damage, total_cycles, remaining_life_pct, threshold_exceeded, created_at
		 FROM snapshots WHERE id = ?`, id,
	)
	var snap model.Snapshot
	var thresholdExceeded int
	var created string
	if err := row.Scan(&snap.ID, &snap.TrialID, &snap.Version, &snap.Status, &snap.TotalDamage, &snap.TotalCycles, &snap.RemainingLifePct, &thresholdExceeded, &created); err != nil {
		return nil, err
	}
	snap.ThresholdExceeded = thresholdExceeded != 0
	snap.CreatedAt = parseTime(created)
	return &snap, nil
}

// ListByTrial 列出某试验的全部快照（按版本升序）。
func (s *SnapshotStore) ListByTrial(trialID int64) ([]*model.Snapshot, error) {
	rows, err := s.db.Query(
		`SELECT id, trial_id, version, status, total_damage, total_cycles, remaining_life_pct, threshold_exceeded, created_at
		 FROM snapshots WHERE trial_id = ? ORDER BY version ASC`, trialID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		var snap model.Snapshot
		var thresholdExceeded int
		var created string
		if err := rows.Scan(&snap.ID, &snap.TrialID, &snap.Version, &snap.Status, &snap.TotalDamage, &snap.TotalCycles, &snap.RemainingLifePct, &thresholdExceeded, &created); err != nil {
			return nil, err
		}
		snap.ThresholdExceeded = thresholdExceeded != 0
		snap.CreatedAt = parseTime(created)
		out = append(out, &snap)
	}
	return out, rows.Err()
}

// UpdateStatus 更新快照状态（发布/替代）。
func (s *SnapshotStore) UpdateStatus(id int64, status model.SnapshotStatus) error {
	_, err := s.db.Exec(`UPDATE snapshots SET status = ? WHERE id = ?`, string(status), id)
	return err
}

// SupersedePublished 将某试验已发布快照标记为替代。
func (s *SnapshotStore) SupersedePublished(trialID int64) error {
	_, err := s.db.Exec(
		`UPDATE snapshots SET status = ? WHERE trial_id = ? AND status = ?`,
		string(model.SnapshotSuperseded), trialID, string(model.SnapshotPublished),
	)
	return err
}
