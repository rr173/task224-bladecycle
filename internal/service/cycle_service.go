package service

import (
	"database/sql"

	"task224-bladecycle/internal/damage"
	"task224-bladecycle/internal/material"
	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/rainflow"
	"task224-bladecycle/internal/store"
)

// AnalyzeResult 是一次循环计数分析的结果汇总。
type AnalyzeResult struct {
	TrialID        int64   `json:"trial_id"`
	Channels       int     `json:"channels"`
	FullCycles     int     `json:"full_cycles"`
	HalfCycles     int     `json:"half_cycles"`
	TotalCycles    float64 `json:"total_cycles"`
	TotalDamage    float64 `json:"total_damage"`
	MaxAmplitude   float64 `json:"max_amplitude"`
	ThresholdMet   bool    `json:"threshold_met"`
}

// CycleService 负责雨流循环计数与 Miner 损伤累计。
type CycleService struct {
	db     *sql.DB
	tele   *store.TelemetryStore
	cycles *store.CycleStore
	damage *store.DamageStore
	trials *store.TrialStore
	mats   *store.MaterialStore
}

func NewCycleService(db *sql.DB) *CycleService {
	return &CycleService{
		db:     db,
		tele:   store.NewTelemetryStore(db),
		cycles: store.NewCycleStore(db),
		damage: store.NewDamageStore(db),
		trials: store.NewTrialStore(db),
		mats:   store.NewMaterialStore(db),
	}
}

// Analyze 对试验执行循环计数与损伤累计。
//
// 前置：试验处于 analyzing（由 FinishAcquisition 推进）。
// 流程：遍历每个通道的有效遥测段 → 拼接峰谷序列 → 雨流计数 → 写循环 →
// 按材料 S-N 曲线累计 Miner 损伤 → 写损伤记录。
//
// 清理旧结果与写入新结果同处一个事务：若重算在写入过程中失败，
// 整个事务回滚（含旧结果清理），从而保留上一轮完整分析结果。
func (s *CycleService) Analyze(trial *model.Trial) (*AnalyzeResult, error) {
	if trial.Status != model.TrialAnalyzing {
		return nil, model.NewInvalidState("trial %d must be analyzing, got %s", trial.ID, trial.Status)
	}
	mat, err := s.mats.Get(trial.MaterialBatchID)
	if err == sql.ErrNoRows {
		return nil, model.NewNotFound("material batch %d not found", trial.MaterialBatchID)
	}
	if err != nil {
		return nil, err
	}
	curve := material.SNCurve{
		FatigueStrengthCoef: mat.FatigueStrengthCoef,
		FatigueExponent:     mat.FatigueExponent,
		UltimateStrength:    mat.UltimateStrength,
		MeanStressMethod:    mat.MeanStressMethod,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	// 任何中途失败都回滚事务，避免清空上一轮完整结果。
	defer func() { _ = tx.Rollback() }()

	cyclesTx := store.NewCycleStore(tx)
	damageTx := store.NewDamageStore(tx)

	// 清理旧结果，保证重算幂等（随事务提交生效）。
	if err := cyclesTx.DeleteByTrial(trial.ID); err != nil {
		return nil, err
	}
	if err := damageTx.DeleteByTrial(trial.ID); err != nil {
		return nil, err
	}

	res := &AnalyzeResult{TrialID: trial.ID}
	channels := s.channelIndexes(trial.ID)

	for _, ch := range channels {
		segs, err := s.tele.ValidSegmentsByChannel(trial.ID, ch)
		if err != nil {
			return nil, err
		}
		if len(segs) == 0 {
			continue
		}
		// 拼接该通道峰谷序列。
		var peaks []float64
		var srcIDs []int64
		for _, seg := range segs {
			peaks = append(peaks, seg.StrainPeaks...)
			srcIDs = append(srcIDs, seg.ID)
		}
		r := rainflow.Count(peaks)
		for _, c := range r.Cycles {
			cyc := &model.Cycle{
				TrialID:          trial.ID,
				ChannelIndex:     ch,
				Amplitude:        c.Amplitude,
				Mean:             c.Mean,
				Count:            c.Count,
				Status:           model.CycleCounted,
				SourceSegmentIDs: srcIDs,
			}
			if _, err := cyclesTx.Insert(cyc); err != nil {
				return nil, err
			}
		}
		res.FullCycles += r.FullCycles
		res.HalfCycles += r.HalfCycles
	}

	// 汇总全部循环计算损伤。
	allCycles, err := cyclesTx.ListByTrial(trial.ID)
	if err != nil {
		return nil, err
	}
	rfCycles := make([]rainflow.Cycle, 0, len(allCycles))
	for _, c := range allCycles {
		rfCycles = append(rfCycles, rainflow.Cycle{
			Amplitude: c.Amplitude,
			Mean:      c.Mean,
			Count:     c.Count,
		})
		if c.Amplitude > res.MaxAmplitude {
			res.MaxAmplitude = c.Amplitude
		}
	}
	totalDamage, totalCycles := damage.Accumulate(rfCycles, curve)
	res.TotalCycles = totalCycles
	res.TotalDamage = totalDamage
	res.ThresholdMet = totalDamage >= 1.0

	rec := &model.DamageRecord{
		TrialID:         trial.ID,
		MaterialBatchID: trial.MaterialBatchID,
		TotalCycles:     totalCycles,
		TotalDamage:     totalDamage,
		MaxAmplitude:    res.MaxAmplitude,
		ThresholdMet:    res.ThresholdMet,
	}
	if _, err := damageTx.Insert(rec); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return res, nil
}

// ListCycles 列出某试验的全部循环（按幅值降序）。
func (s *CycleService) ListCycles(trialID int64) ([]*model.Cycle, error) {
	return s.cycles.ListByTrial(trialID)
}

// LatestDamage 返回某试验最新损伤记录。
func (s *CycleService) LatestDamage(trialID int64) (*model.DamageRecord, error) {
	r, err := s.damage.LatestByTrial(trialID)
	if err == sql.ErrNoRows {
		return nil, model.NewNotFound("no damage record for trial %d", trialID)
	}
	return r, err
}

// DamageHistory 返回某试验全部损伤记录（按时间升序）。
func (s *CycleService) DamageHistory(trialID int64) ([]*model.DamageRecord, error) {
	return s.damage.ListByTrial(trialID)
}

// channelIndexes 返回某试验已注册的通道索引列表（升序）。
func (s *CycleService) channelIndexes(trialID int64) []int {
	segs, err := s.tele.ListByTrial(trialID)
	if err != nil {
		return nil
	}
	seen := map[int]bool{}
	var out []int
	for _, seg := range segs {
		if !seen[seg.ChannelIndex] {
			seen[seg.ChannelIndex] = true
			out = append(out, seg.ChannelIndex)
		}
	}
	// 冒泡排序保证稳定升序。
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
