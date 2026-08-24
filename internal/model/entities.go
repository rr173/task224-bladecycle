// Package model 定义航空发动机叶片疲劳循环计数服务的核心实体、
// 状态机与错误码。所有实体在进入持久化层前必须通过本包的状态校验。
package model

import (
	"time"
)

// 试验架次生命周期状态。
type TrialStatus string

const (
	TrialReady      TrialStatus = "ready"      // 准备
	TrialRunning    TrialStatus = "running"    // 运行中（接收遥测）
	TrialAnalyzing  TrialStatus = "analyzing"  // 分析中（执行循环计数）
	TrialConfirmed  TrialStatus = "confirmed"  // 已确认（损伤结论已复核）
	TrialSealed     TrialStatus = "sealed"     // 封存（单向终态）
)

// 遥测段状态。
type SegmentStatus string

const (
	SegmentPending   SegmentStatus = "pending"   // 待排序/待校验
	SegmentValid     SegmentStatus = "valid"     // 有效
	SegmentGap       SegmentStatus = "gap"       // 存在缺口（不参与计数但保留）
	SegmentDrift     SegmentStatus = "drift"     // 传感器漂移（被标记，不参与计数）
	SegmentDuplicate SegmentStatus = "duplicate" // 重复（幂等键拦截）
)

// 疲劳循环状态。
type CycleStatus string

const (
	CycleCandidate CycleStatus = "candidate" // 候选
	CycleCounted   CycleStatus = "counted"   // 已计数
	CycleReview    CycleStatus = "review"    // 待复核
	CycleConfirmed CycleStatus = "confirmed" // 确认
)

// 寿命快照状态。
type SnapshotStatus string

const (
	SnapshotDraft      SnapshotStatus = "draft"      // 草稿
	SnapshotPublished  SnapshotStatus = "published"  // 发布
	SnapshotSuperseded SnapshotStatus = "superseded" // 替代
)

// Trial 表示一次航空发动机叶片疲劳试验架次。
type Trial struct {
	ID              int64       `json:"id"`
	Name            string      `json:"name"`
	EngineModel     string      `json:"engine_model"`
	MaterialBatchID int64       `json:"material_batch_id"`
	SampleRateHz    float64     `json:"sample_rate_hz"`
	Status          TrialStatus `json:"status"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// Material 表示一个材料批次及其 S-N 曲线（Basquin）参数。
type Material struct {
	ID                  int64     `json:"id"`
	Code                string    `json:"code"`
	Name                string    `json:"name"`
	FatigueStrengthCoef float64   `json:"fatigue_strength_coef"` // σ_f' (MPa)
	FatigueExponent     float64   `json:"fatigue_exponent"`      // b（负值）
	UltimateStrength    float64   `json:"ultimate_strength"`     // σ_u (MPa)
	MeanStressMethod    string    `json:"mean_stress_method"`    // goodman | morrow
	CreatedAt           time.Time `json:"created_at"`
}

// Channel 表示一个遥测通道（叶片应变片）。
type Channel struct {
	ID          int64     `json:"id"`
	TrialID     int64     `json:"trial_id"`
	ChannelIndex int      `json:"channel_index"`
	SensorID    string    `json:"sensor_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// TelemetrySegment 表示一段遥测：转速、温度与应变峰谷序列摘要。
// 原始应变序列经峰谷提取后存为 StrainPeaksJSON，用于雨流计数。
type TelemetrySegment struct {
	ID              int64         `json:"id"`
	TrialID         int64         `json:"trial_id"`
	ChannelIndex    int           `json:"channel_index"`
	SeqNo           int64         `json:"seq_no"`
	RPM             float64       `json:"rpm"`
	Temperature     float64       `json:"temperature"`
	StrainPeaks     []float64     `json:"strain_peaks"`
	Status          SegmentStatus `json:"status"`
	DriftReason     string        `json:"drift_reason,omitempty"`
	GapNote         string        `json:"gap_note,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
}

// Cycle 表示雨流计数得到的单个疲劳循环（半循环或整循环）。
type Cycle struct {
	ID              int64       `json:"id"`
	TrialID         int64       `json:"trial_id"`
	ChannelIndex    int         `json:"channel_index"`
	Amplitude       float64     `json:"amplitude"` // 应力/应变幅（半幅）
	Mean            float64     `json:"mean"`      // 平均应力
	Count           float64     `json:"count"`     // 循环计数（0.5/1）
	Status          CycleStatus `json:"status"`
	SourceSegmentIDs []int64    `json:"source_segment_ids,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
}

// DamageRecord 表示按材料批次累计的 Miner 损伤快照。
type DamageRecord struct {
	ID              int64     `json:"id"`
	TrialID         int64     `json:"trial_id"`
	MaterialBatchID int64     `json:"material_batch_id"`
	TotalCycles     float64   `json:"total_cycles"`
	TotalDamage     float64   `json:"total_damage"`
	MaxAmplitude    float64   `json:"max_amplitude"`
	ThresholdMet    bool      `json:"threshold_met"`
	CreatedAt       time.Time `json:"created_at"`
}

// Snapshot 表示一次寿命快照，用于冻结某时刻的损伤评估结论。
type Snapshot struct {
	ID                int64          `json:"id"`
	TrialID           int64          `json:"trial_id"`
	Version           int            `json:"version"`
	Status            SnapshotStatus `json:"status"`
	TotalDamage       float64        `json:"total_damage"`
	TotalCycles       float64        `json:"total_cycles"`
	RemainingLifePct  float64        `json:"remaining_life_pct"`
	ThresholdExceeded bool           `json:"threshold_exceeded"`
	CreatedAt         time.Time      `json:"created_at"`
}

// CycleResult 是一次雨流计数的汇总结果，供分析流程传递。
type CycleResult struct {
	PeakValleys int     `json:"peak_valleys"`
	FullCycles  int     `json:"full_cycles"`
	HalfCycles  int     `json:"half_cycles"`
	Residual    []float64 `json:"residual"`
}
