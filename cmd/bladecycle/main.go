// Command bladecycle 航空发动机叶片疲劳循环计数服务入口。
//
// 支持三个标志：
//   - --addr :8080      监听地址（默认 :8080）
//   - --db ./bladecycle.db  SQLite 数据库路径（默认 ./bladecycle.db）
//   - --smoke-test      执行端到端冒烟：真实创建材料/试验/遥测、雨流循环
//     计数、Miner 损伤累计、寿命快照，关闭并重开数据库验证持久化与重启
//     恢复，随后以 0 退出码结束。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"

	"task224-bladecycle/internal/httpapi"
	"task224-bladecycle/internal/service"
	"task224-bladecycle/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "./bladecycle.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(*dbPath); err != nil {
			fmt.Fprintln(os.Stderr, "SMOKE TEST FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("SMOKE TEST PASSED")
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	app, err := service.New(db)
	if err != nil {
		log.Fatalf("init services: %v", err)
	}
	srv := httpapi.New(app)
	log.Printf("task224-bladecycle listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

// 冒烟测试固定参数。
const (
	smokeSampleRate = 1000.0 // Hz
	smokePoints     = 1000   // 每段采样点（= 1 秒）
	smokeAmpLow     = 120.0  // 主循环幅值（MPa）
	smokeFreqLow    = 2.0    // 主循环频率（每段 2 个完整周期）
	smokeSegments   = 20     // 通道 0 正常段数
	smokeDriftSegs  = 5      // 通道 1 漂移段数
)

// genLoad 生成一段叶片应力载荷（MPa）：单一频率主循环 + 噪声。
func genLoad(rng *rand.Rand, n int) []float64 {
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		t := float64(i) / smokeSampleRate
		v := smokeAmpLow * math.Sin(2*math.Pi*smokeFreqLow*t)
		v += rng.NormFloat64() * 3.0
		out[i] = v
	}
	return out
}

// genDriftLoad 生成带线性斜坡漂移的载荷（基线单调漂移 + 噪声，用于漂移检测）。
func genDriftLoad(rng *rand.Rand, n int) []float64 {
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		drift := 300.0 * float64(i) / float64(n)
		v := drift + rng.NormFloat64()*5.0
		out[i] = v
	}
	return out
}

// runSmokeTest 执行端到端冒烟：
//
//  1. 打开数据库 A，走完整闭环：材料 → 试验 → 采集 → 多通道遥测（含漂移段、
//     幂等重放、缺口）→ 分析（雨流计数 + Miner 损伤）→ 快照 → 确认 → 封存；
//  2. 幂等验证：重复 seq_no 被跳过；
//  3. 漂移段被排除出计数；
//  4. 封存后拒绝再接收；
//  5. 关闭数据库 A，重开同一路径数据库 B，验证数据仍在（重启恢复）。
func runSmokeTest(dbPath string) error {
	if dbPath != ":memory:" {
		_ = os.Remove(dbPath)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	app, err := service.New(db)
	if err != nil {
		db.Close()
		return fmt.Errorf("init services: %w", err)
	}

	rng := rand.New(rand.NewSource(42))

	// --- 步骤 1：创建材料（较弱材料便于损伤累积到可观测区间） ---
	mat, err := app.Mats.Create("Ti-Al-600", "钛铝合金试验批次", 200.0, -0.12, 300.0, "goodman")
	if err != nil {
		db.Close()
		return fmt.Errorf("create material: %w", err)
	}

	// --- 步骤 2：创建试验并开始采集 ---
	trial, err := app.Trials.Create("发动机叶片疲劳试验 A", "TF-2000", mat.ID, smokeSampleRate)
	if err != nil {
		db.Close()
		return fmt.Errorf("create trial: %w", err)
	}
	trial, err = app.Trials.Start(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("start trial: %w", err)
	}

	// --- 步骤 3：通道 0 接收正常载荷段 ---
	for seq := int64(1); seq <= smokeSegments; seq++ {
		load := genLoad(rng, smokePoints)
		if _, err := app.Tele.Ingest(context.Background(), trial, 0, "strain-gauge-0", seq, 12000, 650, load); err != nil {
			db.Close()
			return fmt.Errorf("ingest ch0 seq=%d: %w", seq, err)
		}
	}

	// --- 步骤 4：通道 1 接收漂移段（应被标记为 drift） ---
	for seq := int64(1); seq <= smokeDriftSegs; seq++ {
		load := genDriftLoad(rng, smokePoints)
		if _, err := app.Tele.Ingest(context.Background(), trial, 1, "strain-gauge-1", seq, 12000, 650, load); err != nil {
			db.Close()
			return fmt.Errorf("ingest ch1 seq=%d: %w", seq, err)
		}
	}

	// --- 步骤 5：幂等验证：重复 seq_no 被跳过 ---
	res, err := app.Tele.Ingest(context.Background(), trial, 0, "strain-gauge-0", 10, 12000, 650, genLoad(rng, smokePoints))
	if err != nil {
		db.Close()
		return fmt.Errorf("re-ingest: %w", err)
	}
	if res.Duplicate != 1 || res.Inserted != 0 {
		db.Close()
		return fmt.Errorf("re-ingest should be duplicate, got inserted=%d duplicate=%d", res.Inserted, res.Duplicate)
	}

	// --- 步骤 6：缺口验证：通道 0 seq 22（跳过 21）触发 gap ---
	gapRes, err := app.Tele.Ingest(context.Background(), trial, 0, "strain-gauge-0", 22, 12000, 650, genLoad(rng, smokePoints))
	if err != nil {
		db.Close()
		return fmt.Errorf("ingest gap: %w", err)
	}
	if !gapRes.Gap {
		db.Close()
		return fmt.Errorf("expected gap flag on seq=22, got gap=%v", gapRes.Gap)
	}

	// --- 步骤 7：漂移段计数验证 ---
	driftCount, err := app.Tele.CountDrift(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("count drift: %w", err)
	}
	if driftCount != smokeDriftSegs {
		db.Close()
		return fmt.Errorf("expected %d drift segments, got %d", smokeDriftSegs, driftCount)
	}

	// --- 步骤 8：结束采集并分析 ---
	trial, err = app.Trials.FinishAcquisition(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("finish acquisition: %w", err)
	}
	analyzeRes, err := app.Cycles.Analyze(trial)
	if err != nil {
		db.Close()
		return fmt.Errorf("analyze: %w", err)
	}
	if analyzeRes.FullCycles == 0 && analyzeRes.HalfCycles == 0 {
		db.Close()
		return fmt.Errorf("no fatigue cycles extracted")
	}
	if analyzeRes.TotalDamage <= 0 {
		db.Close()
		return fmt.Errorf("expected positive damage, got %f", analyzeRes.TotalDamage)
	}

	// 漂移段被排除：通道 1 无循环。
	cycles, err := app.Cycles.ListCycles(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("list cycles: %w", err)
	}
	for _, c := range cycles {
		if c.ChannelIndex == 1 {
			db.Close()
			return fmt.Errorf("drift channel should not contribute cycles, found channel_index=1")
		}
	}

	// --- 步骤 9：发布寿命快照 ---
	snap, err := app.Snaps.Publish(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("publish snapshot: %w", err)
	}
	if snap.RemainingLifePct <= 0 || snap.RemainingLifePct > 100 {
		db.Close()
		return fmt.Errorf("remaining life pct out of range: %f", snap.RemainingLifePct)
	}

	// --- 步骤 10：确认并封存 ---
	trial, err = app.Trials.Confirm(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("confirm: %w", err)
	}
	trial, err = app.Trials.Seal(trial.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("seal: %w", err)
	}

	// --- 步骤 11：封存后拒绝再接收 ---
	_, err = app.Tele.Ingest(context.Background(), trial, 0, "strain-gauge-0", 23, 12000, 650, genLoad(rng, smokePoints))
	if err == nil {
		db.Close()
		return fmt.Errorf("expected sealed trial to reject ingest")
	}

	// --- 步骤 12：记录关键值用于重启比对 ---
	sealedStatus := trial.Status
	cycleCount := len(cycles)
	snapshotVersion := snap.Version
	snapshotDamage := snap.TotalDamage

	db.Close()

	// --- 步骤 13：重开同一数据库，验证持久化与重启恢复 ---
	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	defer db2.Close()
	app2, err := service.New(db2)
	if err != nil {
		return fmt.Errorf("reinit services: %w", err)
	}
	reloaded, err := app2.Trials.Get(trial.ID)
	if err != nil {
		return fmt.Errorf("reload trial: %w", err)
	}
	if reloaded.Status != sealedStatus {
		return fmt.Errorf("recovered trial status %s != sealed %s", reloaded.Status, sealedStatus)
	}
	cycles2, err := app2.Cycles.ListCycles(trial.ID)
	if err != nil {
		return fmt.Errorf("reload cycles: %w", err)
	}
	if len(cycles2) != cycleCount {
		return fmt.Errorf("recovered cycles %d != %d", len(cycles2), cycleCount)
	}
	snapshots, err := app2.Snaps.ListByTrial(trial.ID)
	if err != nil {
		return fmt.Errorf("reload snapshots: %w", err)
	}
	if len(snapshots) != 1 || snapshots[0].Version != snapshotVersion {
		return fmt.Errorf("recovered snapshot mismatch: %+v", snapshots)
	}
	if math.Abs(snapshots[0].TotalDamage-snapshotDamage) > 1e-9 {
		return fmt.Errorf("recovered snapshot damage %f != %f", snapshots[0].TotalDamage, snapshotDamage)
	}

	// 记录冒烟结果摘要。
	fmt.Printf("SMOKE OK: trial=%s cycles=%d full=%d half=%d damage=%.6f remaining=%.1f%% snapshot=v%d\n",
		reloaded.Status, cycleCount, analyzeRes.FullCycles, analyzeRes.HalfCycles,
		snapshotDamage, snap.RemainingLifePct, snapshotVersion)
	return nil
}
