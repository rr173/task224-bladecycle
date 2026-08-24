package store

import (
	"database/sql"

	"task224-bladecycle/internal/model"
)

// CycleStore 管理疲劳循环（雨流计数结果）的持久化。
type CycleStore struct {
	db *sql.DB
}

func NewCycleStore(db *sql.DB) *CycleStore { return &CycleStore{db: db} }

// Insert 插入单个疲劳循环。
func (s *CycleStore) Insert(c *model.Cycle) (int64, error) {
	idsJSON, err := encodeIDs(c.SourceSegmentIDs)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(
		`INSERT INTO cycles (trial_id, channel_index, amplitude, mean, count, status, source_segment_ids_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.TrialID, c.ChannelIndex, c.Amplitude, c.Mean, c.Count, string(c.Status), idsJSON, nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListByTrial 列出某试验的全部疲劳循环。
func (s *CycleStore) ListByTrial(trialID int64) ([]*model.Cycle, error) {
	rows, err := s.db.Query(
		`SELECT id, trial_id, channel_index, amplitude, mean, count, status, source_segment_ids_json, created_at
		 FROM cycles WHERE trial_id = ? ORDER BY amplitude DESC`, trialID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Cycle
	for rows.Next() {
		var c model.Cycle
		var idsJSON, created string
		if err := rows.Scan(&c.ID, &c.TrialID, &c.ChannelIndex, &c.Amplitude, &c.Mean, &c.Count, &c.Status, &idsJSON, &created); err != nil {
			return nil, err
		}
		ids, err := decodeIDs(idsJSON)
		if err != nil {
			return nil, err
		}
		c.SourceSegmentIDs = ids
		c.CreatedAt = parseTime(created)
		out = append(out, &c)
	}
	return out, rows.Err()
}

// SumCycles 统计某试验的总循环计数。
func (s *CycleStore) SumCycles(trialID int64) (float64, error) {
	var sum sql.NullFloat64
	err := s.db.QueryRow(
		`SELECT SUM(count) FROM cycles WHERE trial_id = ?`, trialID,
	).Scan(&sum)
	if !sum.Valid {
		return 0, nil
	}
	return sum.Float64, err
}

// CountByStatus 统计某试验指定状态的循环数。
func (s *CycleStore) CountByStatus(trialID int64, status model.CycleStatus) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM cycles WHERE trial_id = ? AND status = ?`,
		trialID, string(status),
	).Scan(&n)
	return n, err
}

// UpdateStatus 更新循环状态。
func (s *CycleStore) UpdateStatus(id int64, status model.CycleStatus) error {
	_, err := s.db.Exec(
		`UPDATE cycles SET status = ? WHERE id = ?`, string(status), id,
	)
	return err
}

// DeleteByTrial 删除某试验的全部循环（重算前清理）。
func (s *CycleStore) DeleteByTrial(trialID int64) error {
	_, err := s.db.Exec(`DELETE FROM cycles WHERE trial_id = ?`, trialID)
	return err
}
