// Package store 基于 SQLite（modernc.org/sqlite，纯 Go 驱动）提供持久化。
// 所有建表迁移集中在 migrate，CRUD 分布在 *_store.go，共享一个 *sql.DB。
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，CGO 无关，离线可构建
)

// dsn 将原始路径/URI 转为带连接级配置的 DSN。
//
// 通过驱动查询参数对连接池中的每条连接统一配置，避免 PRAGMA 仅作用于单条
// 连接、其余连接无超时配置的隐患：
//   - _txlock=immediate：写事务在 BEGIN 时即获取 RESERVED 锁，把对同一试验
//     的并发发布请求串行化（先拿到锁的事务完整执行取版本号→替代旧快照→写入，
//     其余事务在 busy_timeout 内排队），消除交错执行导致的版本号竞争与重复版本；
//   - _pragma=busy_timeout(...)：SQLITE_BUSY 时由 SQLite 内核在超时窗口内自动
//     重试，配合上述串行化让并发发布请求相互等待而非直接失败。
//
// file: URI 自带查询串时原样交由驱动处理；其余路径追加参数（驱动仅在 DSN 含
// '?' 时解析查询参数，故对纯文件路径安全）。
func dsn(path string) string {
	if path == "" {
		path = ":memory:"
	}
	if len(path) >= 5 && path[:5] == "file:" {
		return path
	}
	sep := "?"
	if containsRune(path, '?') {
		sep = "&"
	}
	return path + sep + "_txlock=immediate&_pragma=busy_timeout(5000)"
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// Open 打开（必要时创建）SQLite 数据库并执行建表迁移。
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// WAL 与外键需在迁移前设置；busy_timeout 已经由 DSN 对每条连接统一配置。
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// migrate 创建全部表；语句均为幂等 CREATE TABLE IF NOT EXISTS。
func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			engine_model TEXT NOT NULL,
			material_batch_id INTEGER NOT NULL,
			sample_rate_hz REAL NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS materials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			fatigue_strength_coef REAL NOT NULL,
			fatigue_exponent REAL NOT NULL,
			ultimate_strength REAL NOT NULL,
			mean_stress_method TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			channel_index INTEGER NOT NULL,
			sensor_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, channel_index)
		)`,
		`CREATE TABLE IF NOT EXISTS telemetry_segments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			channel_index INTEGER NOT NULL,
			seq_no INTEGER NOT NULL,
			rpm REAL NOT NULL,
			temperature REAL NOT NULL,
			strain_peaks_json TEXT NOT NULL,
			status TEXT NOT NULL,
			drift_reason TEXT NOT NULL DEFAULT '',
			gap_note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, channel_index, seq_no)
		)`,
		`CREATE TABLE IF NOT EXISTS cycles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			channel_index INTEGER NOT NULL,
			amplitude REAL NOT NULL,
			mean REAL NOT NULL,
			count REAL NOT NULL,
			status TEXT NOT NULL,
			source_segment_ids_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS damage_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			material_batch_id INTEGER NOT NULL,
			total_cycles REAL NOT NULL,
			total_damage REAL NOT NULL,
			max_amplitude REAL NOT NULL,
			threshold_met INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			trial_id INTEGER NOT NULL,
			version INTEGER NOT NULL,
			status TEXT NOT NULL,
			total_damage REAL NOT NULL,
			total_cycles REAL NOT NULL,
			remaining_life_pct REAL NOT NULL,
			threshold_exceeded INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(trial_id, version)
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
	}
	return nil
}
