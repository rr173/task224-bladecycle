package store

import (
	"database/sql"

	"task224-bladecycle/internal/model"
)

// TrialStore 管理试验架次的持久化。
type TrialStore struct {
	db *sql.DB
}

func NewTrialStore(db *sql.DB) *TrialStore { return &TrialStore{db: db} }

// Insert 创建试验架次，返回自增 ID。
func (s *TrialStore) Insert(name, engineModel string, materialBatchID int64, sampleRateHz float64) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO trials (name, engine_model, material_batch_id, sample_rate_hz, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		name, engineModel, materialBatchID, sampleRateHz, string(model.TrialReady), nowStr(), nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Get 按 ID 查询试验架次。
func (s *TrialStore) Get(id int64) (*model.Trial, error) {
	row := s.db.QueryRow(
		`SELECT id, name, engine_model, material_batch_id, sample_rate_hz, status, created_at, updated_at
		 FROM trials WHERE id = ?`, id,
	)
	var t model.Trial
	var created, updated string
	if err := row.Scan(&t.ID, &t.Name, &t.EngineModel, &t.MaterialBatchID, &t.SampleRateHz, &t.Status, &created, &updated); err != nil {
		return nil, err
	}
	t.CreatedAt = parseTime(created)
	t.UpdatedAt = parseTime(updated)
	return &t, nil
}

// List 列出全部试验架次（按 ID 降序）。
func (s *TrialStore) List() ([]*model.Trial, error) {
	rows, err := s.db.Query(
		`SELECT id, name, engine_model, material_batch_id, sample_rate_hz, status, created_at, updated_at
		 FROM trials ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Trial
	for rows.Next() {
		var t model.Trial
		var created, updated string
		if err := rows.Scan(&t.ID, &t.Name, &t.EngineModel, &t.MaterialBatchID, &t.SampleRateHz, &t.Status, &created, &updated); err != nil {
			return nil, err
		}
		t.CreatedAt = parseTime(created)
		t.UpdatedAt = parseTime(updated)
		out = append(out, &t)
	}
	return out, rows.Err()
}

// UpdateStatus 更新试验状态（状态机校验在上层 service 完成）。
func (s *TrialStore) UpdateStatus(id int64, status model.TrialStatus) error {
	_, err := s.db.Exec(
		`UPDATE trials SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), nowStr(), id,
	)
	return err
}
