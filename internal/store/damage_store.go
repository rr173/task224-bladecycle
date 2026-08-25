package store

import (
	"task224-bladecycle/internal/model"
)

// DamageStore 管理 Miner 损伤累计记录的持久化。
type DamageStore struct {
	db DBTX
}

func NewDamageStore(db DBTX) *DamageStore { return &DamageStore{db: db} }

// Insert 写入一条损伤累计记录。
func (s *DamageStore) Insert(r *model.DamageRecord) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO damage_records (trial_id, material_batch_id, total_cycles, total_damage, max_amplitude, threshold_met, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.TrialID, r.MaterialBatchID, r.TotalCycles, r.TotalDamage, r.MaxAmplitude, boolToInt(r.ThresholdMet), nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListByTrial 列出某试验的损伤记录（按时间升序）。
func (s *DamageStore) ListByTrial(trialID int64) ([]*model.DamageRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, trial_id, material_batch_id, total_cycles, total_damage, max_amplitude, threshold_met, created_at
		 FROM damage_records WHERE trial_id = ? ORDER BY id ASC`, trialID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.DamageRecord
	for rows.Next() {
		var r model.DamageRecord
		var thresholdMet int
		var created string
		if err := rows.Scan(&r.ID, &r.TrialID, &r.MaterialBatchID, &r.TotalCycles, &r.TotalDamage, &r.MaxAmplitude, &thresholdMet, &created); err != nil {
			return nil, err
		}
		r.ThresholdMet = thresholdMet != 0
		r.CreatedAt = parseTime(created)
		out = append(out, &r)
	}
	return out, rows.Err()
}

// LatestByTrial 返回某试验最新一条损伤记录。
func (s *DamageStore) LatestByTrial(trialID int64) (*model.DamageRecord, error) {
	row := s.db.QueryRow(
		`SELECT id, trial_id, material_batch_id, total_cycles, total_damage, max_amplitude, threshold_met, created_at
		 FROM damage_records WHERE trial_id = ? ORDER BY id DESC LIMIT 1`, trialID,
	)
	var r model.DamageRecord
	var thresholdMet int
	var created string
	if err := row.Scan(&r.ID, &r.TrialID, &r.MaterialBatchID, &r.TotalCycles, &r.TotalDamage, &r.MaxAmplitude, &thresholdMet, &created); err != nil {
		return nil, err
	}
	r.ThresholdMet = thresholdMet != 0
	r.CreatedAt = parseTime(created)
	return &r, nil
}

// DeleteByTrial 删除某试验的损伤记录（重算前清理）。
func (s *DamageStore) DeleteByTrial(trialID int64) error {
	_, err := s.db.Exec(`DELETE FROM damage_records WHERE trial_id = ?`, trialID)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
