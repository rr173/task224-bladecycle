package store

import (
	"database/sql"
	"errors"
	"sync"

	"task224-bladecycle/internal/model"
)

// SnapshotStore 管理寿命快照（冻结损伤评估结论）的持久化。
type SnapshotStore struct {
	db *sql.DB
	// publishMu 串行化发布流程，保证进程内同一时刻只有一个发布事务在运行。
	// 配合事务内 MAX(version)+INSERT 的原子性，杜绝并发发布导致的版本号竞争与
	// 重复版本；同时避免多连接争抢 RESERVED 锁造成 SQLITE_BUSY 堆积。
	publishMu sync.Mutex
}

func NewSnapshotStore(db *sql.DB) *SnapshotStore { return &SnapshotStore{db: db} }

// SnapshotFreezer 在持锁事务内计算待发布的快照内容。
// 它接收该试验最新损伤记录，返回冻结后的损伤/循环/剩余寿命等字段（不含 Version/ID）。
//
// 将冻结计算放在回调里，可保证读损伤、分配版本号、替代旧快照、写入新快照
// 全部位于同一事务中，互不交错。
type SnapshotFreezer func(rec *model.DamageRecord) *model.Snapshot

// maxPublishAttempts 限定并发版本冲突时的重试上限。
// 正常情况下 publishMu 已串行化发布，冲突仅在极端竞态下出现。
const maxPublishAttempts = 8

// Publish 原子地发布一条寿命快照。
//
// publishMu 在进程内串行化发布；事务（_txlock=immediate 提供 RESERVED 锁）
// 负责把读损伤、分配版本号、替代旧快照、写入新快照作为一个原子单元：
//  1. 读取该试验最新损伤记录，无记录则返回 sql.ErrNoRows；
//  2. 由 freezer 计算冻结内容；
//  3. 取下一个版本号（MAX(version)+1）；
//  4. 将已发布的旧快照标记为 superseded；
//  5. 写入新快照（UNIQUE(trial_id, version)）。
//
// 双重保障下，同一试验短时间多次发布请求不会交错执行；万一仍命中
// UNIQUE(trial_id, version) 冲突，重试整个读-分配-写序列，确保调用方拿到
// 完整且唯一的版本结果。
func (s *SnapshotStore) Publish(trialID int64, freezer SnapshotFreezer) (*model.Snapshot, error) {
	s.publishMu.Lock()
	defer s.publishMu.Unlock()

	var lastErr error
	for attempt := 0; attempt < maxPublishAttempts; attempt++ {
		snap, err := s.publishOnce(trialID, freezer)
		if err == nil {
			return snap, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		lastErr = err
		if !isUniqueViolation(err) {
			return nil, err
		}
		// UNIQUE(trial_id, version) 冲突 → 重新读取版本号重试。
	}
	if lastErr == nil {
		lastErr = errors.New("publish snapshot: exhausted retries")
	}
	return nil, lastErr
}

// publishOnce 执行一次完整的发布事务。
func (s *SnapshotStore) publishOnce(trialID int64, freezer SnapshotFreezer) (*model.Snapshot, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // 提交后为 no-op。

	rec, err := latestDamageInTx(tx, trialID)
	if err != nil {
		return nil, err
	}

	snap := freezer(rec)
	snap.TrialID = trialID

	version, err := nextVersionInTx(tx, trialID)
	if err != nil {
		return nil, err
	}
	snap.Version = version
	snap.Status = model.SnapshotPublished

	if err := supersedePublishedInTx(tx, trialID); err != nil {
		return nil, err
	}
	id, err := insertSnapshotInTx(tx, snap)
	if err != nil {
		return nil, err
	}
	snap.ID = id

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return snap, nil
}

// NextVersion 返回某试验下一个快照版本号（从 1 开始递增）。
//
// 保留供非发布场景使用；发布流程请改用 Publish 以获得事务原子性。
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

// nextVersionInTx 在事务内读取下一个版本号，保证与写入不可分割。
func nextVersionInTx(tx *sql.Tx, trialID int64) (int, error) {
	var v sql.NullInt64
	if err := tx.QueryRow(
		`SELECT MAX(version) FROM snapshots WHERE trial_id = ?`, trialID,
	).Scan(&v); err != nil {
		return 0, err
	}
	if !v.Valid {
		return 1, nil
	}
	return int(v.Int64) + 1, nil
}

// Insert 写入一条寿命快照。UNIQUE(trial_id, version) 保证版本唯一。
//
// 保留供非发布场景直接写入；发布流程走 Publish 以获得事务保护。
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

// insertSnapshotInTx 在事务内写入快照，便于回滚。
func insertSnapshotInTx(tx *sql.Tx, snap *model.Snapshot) (int64, error) {
	res, err := tx.Exec(
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

// latestDamageInTx 在事务内读取某试验最新损伤记录，无记录返回 sql.ErrNoRows。
func latestDamageInTx(tx *sql.Tx, trialID int64) (*model.DamageRecord, error) {
	row := tx.QueryRow(
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

// supersedePublishedInTx 在事务内将已发布快照标记为替代。
func supersedePublishedInTx(tx *sql.Tx, trialID int64) error {
	_, err := tx.Exec(
		`UPDATE snapshots SET status = ? WHERE trial_id = ? AND status = ?`,
		string(model.SnapshotSuperseded), trialID, string(model.SnapshotPublished),
	)
	return err
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
