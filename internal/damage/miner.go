// Package damage 实现 Miner 线性累积损伤法则：将雨流计数得到的每个循环
// 按材料 S-N 曲线折算为损伤贡献并累加，D = Σ (n_i / N_fi)。
package damage

import (
	"math"

	"task224-bladecycle/internal/material"
	"task224-bladecycle/internal/rainflow"
)

// Accumulate 计算一批循环的总损伤与总循环计数。
//
// 对每个循环：先做平均应力修正得到等效应力幅 σ_ar，再查 S-N 曲线得寿命
// N_f，累加 count / N_f。寿命无穷（幅值过低）的循环不贡献损伤。
func Accumulate(cycles []rainflow.Cycle, curve material.SNCurve) (totalDamage float64, totalCycles float64) {
	for _, c := range cycles {
		totalCycles += c.Count
		amp := curve.CorrectAmplitude(c.Amplitude, c.Mean)
		nf := curve.LifeCycles(amp)
		if nf <= 0 || math.IsInf(nf, 1) || math.IsNaN(nf) {
			continue
		}
		totalDamage += c.Count / nf
	}
	return totalDamage, totalCycles
}

// Bin 是损伤按幅值区间的分桶统计。
type Bin struct {
	Lower       float64 `json:"lower"`
	Upper       float64 `json:"upper"`
	CycleCount  float64 `json:"cycle_count"`
	Damage      float64 `json:"damage"`
}

// BinDamage 将循环按幅值区间分桶并计算每桶损伤贡献。
// edges 为区间边界（升序），例如 [0, 200, 400, 600]。
func BinDamage(cycles []rainflow.Cycle, curve material.SNCurve, edges []float64) []Bin {
	bins := make([]Bin, 0, len(edges)-1)
	for i := 0; i+1 < len(edges); i++ {
		bins = append(bins, Bin{Lower: edges[i], Upper: edges[i+1]})
	}
	for _, c := range cycles {
		amp := curve.CorrectAmplitude(c.Amplitude, c.Mean)
		nf := curve.LifeCycles(amp)
		contrib := 0.0
		if nf > 0 && !math.IsInf(nf, 1) && !math.IsNaN(nf) {
			contrib = c.Count / nf
		}
		// 使用原始幅值分桶。
		for i := range bins {
			if c.Amplitude >= bins[i].Lower && (c.Amplitude < bins[i].Upper || i == len(bins)-1) {
				bins[i].CycleCount += c.Count
				bins[i].Damage += contrib
				break
			}
		}
	}
	return bins
}

// RemainingLife 根据总损伤估算剩余寿命百分比（相对损伤阈值 1.0）。
// 返回 0~100；超过阈值返回 0。
func RemainingLife(totalDamage float64) float64 {
	if totalDamage >= 1.0 {
		return 0
	}
	return (1.0 - totalDamage) * 100
}
