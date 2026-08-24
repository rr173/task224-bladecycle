// Package store 基于 SQLite（modernc.org/sqlite，纯 Go 驱动）提供持久化。
// 所有建表迁移集中在 migrate，CRUD 分布在 *_store.go，共享一个 *sql.DB。
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // 纯 Go SQLite 驱动，CGO 无关，离线可构建
)

// Open 打开（必要时创建）SQLite 数据库并执行建表迁移。
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// 单写者 + 外键 + 事务并发稳定。
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
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
