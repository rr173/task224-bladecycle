package store

import (
	"database/sql"

	"task224-bladecycle/internal/model"
)

// MaterialStore 管理材料批次（S-N 曲线参数）的持久化。
type MaterialStore struct {
	db *sql.DB
}

func NewMaterialStore(db *sql.DB) *MaterialStore { return &MaterialStore{db: db} }

// Insert 创建材料批次。
func (s *MaterialStore) Insert(m *model.Material) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO materials (code, name, fatigue_strength_coef, fatigue_exponent, ultimate_strength, mean_stress_method, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		m.Code, m.Name, m.FatigueStrengthCoef, m.FatigueExponent, m.UltimateStrength, m.MeanStressMethod, nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Get 按 ID 查询材料批次。
func (s *MaterialStore) Get(id int64) (*model.Material, error) {
	row := s.db.QueryRow(
		`SELECT id, code, name, fatigue_strength_coef, fatigue_exponent, ultimate_strength, mean_stress_method, created_at
		 FROM materials WHERE id = ?`, id,
	)
	var m model.Material
	var created string
	if err := row.Scan(&m.ID, &m.Code, &m.Name, &m.FatigueStrengthCoef, &m.FatigueExponent, &m.UltimateStrength, &m.MeanStressMethod, &created); err != nil {
		return nil, err
	}
	m.CreatedAt = parseTime(created)
	return &m, nil
}

// List 列出全部材料批次。
func (s *MaterialStore) List() ([]*model.Material, error) {
	rows, err := s.db.Query(
		`SELECT id, code, name, fatigue_strength_coef, fatigue_exponent, ultimate_strength, mean_stress_method, created_at
		 FROM materials ORDER BY id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Material
	for rows.Next() {
		var m model.Material
		var created string
		if err := rows.Scan(&m.ID, &m.Code, &m.Name, &m.FatigueStrengthCoef, &m.FatigueExponent, &m.UltimateStrength, &m.MeanStressMethod, &created); err != nil {
			return nil, err
		}
		m.CreatedAt = parseTime(created)
		out = append(out, &m)
	}
	return out, rows.Err()
}
