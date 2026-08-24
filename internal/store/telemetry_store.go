package store

import (
	"database/sql"
	"fmt"
	"strings"

	"task224-bladecycle/internal/model"
)

// TelemetryStore 管理遥测段与通道的持久化。
type TelemetryStore struct {
	db *sql.DB
}

func NewTelemetryStore(db *sql.DB) *TelemetryStore { return &TelemetryStore{db: db} }

// EnsureChannel 幂等注册通道（存在则返回 ID）。
func (s *TelemetryStore) EnsureChannel(trialID int64, channelIndex int, sensorID string) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM channels WHERE trial_id = ? AND channel_index = ?`,
		trialID, channelIndex,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}
	res, err := s.db.Exec(
		`INSERT INTO channels (trial_id, channel_index, sensor_id, created_at) VALUES (?, ?, ?, ?)`,
		trialID, channelIndex, sensorID, nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// InsertSegment 插入遥测段。UNIQUE(trial_id, channel_index, seq_no) 保证幂等，
// 冲突时返回 ErrDuplicate。
func (s *TelemetryStore) InsertSegment(seg *model.TelemetrySegment) (int64, error) {
	peaks, err := encodePeaks(seg.StrainPeaks)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(
		`INSERT INTO telemetry_segments (trial_id, channel_index, seq_no, rpm, temperature, strain_peaks_json, status, drift_reason, gap_note, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		seg.TrialID, seg.ChannelIndex, seg.SeqNo, seg.RPM, seg.Temperature, peaks,
		string(seg.Status), seg.DriftReason, seg.GapNote, nowStr(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, model.ErrDuplicate
		}
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateStatus 更新遥测段状态（可携带漂移原因/缺口说明）。
func (s *TelemetryStore) UpdateStatus(id int64, status model.SegmentStatus, reason, note string) error {
	_, err := s.db.Exec(
		`UPDATE telemetry_segments SET status = ?, drift_reason = ?, gap_note = ? WHERE id = ?`,
		string(status), reason, note, id,
	)
	return err
}

// GetSegment 按 ID 查询遥测段。
func (s *TelemetryStore) GetSegment(id int64) (*model.TelemetrySegment, error) {
	row := s.db.QueryRow(
		`SELECT id, trial_id, channel_index, seq_no, rpm, temperature, strain_peaks_json, status, drift_reason, gap_note, created_at
		 FROM telemetry_segments WHERE id = ?`, id,
	)
	return scanSegment(row)
}

// ListByTrial 列出某试验的全部遥测段（按通道、序号升序）。
func (s *TelemetryStore) ListByTrial(trialID int64) ([]*model.TelemetrySegment, error) {
	rows, err := s.db.Query(
		`SELECT id, trial_id, channel_index, seq_no, rpm, temperature, strain_peaks_json, status, drift_reason, gap_note, created_at
		 FROM telemetry_segments WHERE trial_id = ? ORDER BY channel_index ASC, seq_no ASC`, trialID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.TelemetrySegment
	for rows.Next() {
		seg, err := scanSegment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}

// CountByStatus 统计某试验指定状态的遥测段数量。
func (s *TelemetryStore) CountByStatus(trialID int64, status model.SegmentStatus) (int, error) {
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM telemetry_segments WHERE trial_id = ? AND status = ?`,
		trialID, string(status),
	).Scan(&n)
	return n, err
}

// ValidSegmentsByChannel 返回某通道的有效（可计数）遥测段，按序号升序。
func (s *TelemetryStore) ValidSegmentsByChannel(trialID int64, channelIndex int) ([]*model.TelemetrySegment, error) {
	rows, err := s.db.Query(
		`SELECT id, trial_id, channel_index, seq_no, rpm, temperature, strain_peaks_json, status, drift_reason, gap_note, created_at
		 FROM telemetry_segments WHERE trial_id = ? AND channel_index = ? AND status = ?
		 ORDER BY seq_no ASC`, trialID, channelIndex, string(model.SegmentValid),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.TelemetrySegment
	for rows.Next() {
		seg, err := scanSegment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}

// MaxSeq 返回某通道的最大段序号，无记录时 dst 保持 Invalid。
func (s *TelemetryStore) MaxSeq(trialID int64, channelIndex int, dst *sql.NullInt64) error {
	return s.db.QueryRow(
		`SELECT MAX(seq_no) FROM telemetry_segments WHERE trial_id = ? AND channel_index = ?`,
		trialID, channelIndex,
	).Scan(dst)
}

// scanSegment 从行扫描器读取遥测段。
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSegment(row rowScanner) (*model.TelemetrySegment, error) {
	var seg model.TelemetrySegment
	var peaksJSON, created string
	if err := row.Scan(
		&seg.ID, &seg.TrialID, &seg.ChannelIndex, &seg.SeqNo, &seg.RPM, &seg.Temperature,
		&peaksJSON, &seg.Status, &seg.DriftReason, &seg.GapNote, &created,
	); err != nil {
		return nil, err
	}
	peaks, err := decodePeaks(peaksJSON)
	if err != nil {
		return nil, fmt.Errorf("decode peaks: %w", err)
	}
	seg.StrainPeaks = peaks
	seg.CreatedAt = parseTime(created)
	return &seg, nil
}

// isUniqueViolation 判断 SQLite 唯一约束冲突。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed")
}
