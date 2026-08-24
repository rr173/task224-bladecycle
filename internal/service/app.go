// Package service 是业务编排层：组合 store 与各业务包，实现试验生命周期、
// 遥测清洗、雨流循环计数、Miner 损伤累计与寿命快照的完整闭环。
package service

import (
	"database/sql"

	"task224-bladecycle/internal/store"
)

// App 聚合全部子服务，共享一个 *sql.DB。
type App struct {
	db     *sql.DB
	Trials *TrialService
	Mats   *MaterialService
	Tele   *TelemetryService
	Cycles *CycleService
	Snaps  *SnapshotService
	Stats  *store.StatsStore
}

// New 构造 App 及全部子服务。
func New(db *sql.DB) (*App, error) {
	app := &App{
		db: db,
	}
	app.Trials = NewTrialService(db)
	app.Mats = NewMaterialService(db)
	app.Tele = NewTelemetryService(db)
	app.Cycles = NewCycleService(db)
	app.Snaps = NewSnapshotService(db)
	app.Stats = store.NewStatsStore(db)
	return app, nil
}

// DB 暴露底层数据库（冒烟测试关闭重开用）。
func (a *App) DB() *sql.DB { return a.db }
