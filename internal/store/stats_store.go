package store

import "database/sql"

// Stats 是健康/统计接口的汇总数据。
type Stats struct {
	Trials        int     `json:"trials"`
	Materials     int     `json:"materials"`
	Segments      int     `json:"segments"`
	Cycles        int     `json:"cycles"`
	CyclesSum     float64 `json:"cycles_sum"`
	DamageRecords int     `json:"damage_records"`
	Snapshots     int     `json:"snapshots"`
}

// StatsStore 提供跨表计数统计。
type StatsStore struct {
	db *sql.DB
}

func NewStatsStore(db *sql.DB) *StatsStore { return &StatsStore{db: db} }

// Collect 汇总全库计数。
func (s *StatsStore) Collect() (*Stats, error) {
	st := &Stats{}
	var err error
	if st.Trials, err = countRows(s.db, "trials"); err != nil {
		return nil, err
	}
	if st.Materials, err = countRows(s.db, "materials"); err != nil {
		return nil, err
	}
	if st.Segments, err = countRows(s.db, "telemetry_segments"); err != nil {
		return nil, err
	}
	if st.Cycles, err = countRows(s.db, "cycles"); err != nil {
		return nil, err
	}
	if st.DamageRecords, err = countRows(s.db, "damage_records"); err != nil {
		return nil, err
	}
	if st.Snapshots, err = countRows(s.db, "snapshots"); err != nil {
		return nil, err
	}
	var sum sql.NullFloat64
	if err := s.db.QueryRow("SELECT SUM(count) FROM cycles").Scan(&sum); err == nil && sum.Valid {
		st.CyclesSum = sum.Float64
	}
	return st, nil
}

func countRows(db *sql.DB, table string) (int, error) {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
	return n, err
}
