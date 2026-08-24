// Package rainflow 实现雨流循环计数（rainflow counting）算法。
//
// 采用经典四点法：对连续四个峰谷点 S1,S2,S3,S4，若内距 |S3-S2| 同时
// 不超过两侧距 |S2-S1| 与 |S4-S3|，则 S2→S3 构成一个完整循环，从序列中
// 移除该两点；重复直至无完整循环可提取，剩余点为残差序列，按"腿数守恒"
// 配对：每 2 条腿构成一个完整循环，余 1 条腿构成一个半循环。
package rainflow

import (
	"math"
	"sort"
)

// Cycle 是雨流计数得到的一个循环。
type Cycle struct {
	Amplitude float64 // 半幅 = |峰-谷| / 2
	Mean      float64 // 平均应力/应变
	Count     float64 // 1.0 = 完整循环；0.5 = 半循环
}

// Result 是一次雨流计数的完整结果。
type Result struct {
	Cycles     []Cycle   // 全部循环（含半循环）
	Residual   []float64 // 残差峰谷序列
	FullCycles int       // 四点法提取的完整循环数
	HalfCycles int       // 残差配对产生的半循环数
}

// Count 对峰谷序列执行雨流计数。
//
// 输入必须已经是峰谷序列（可先经 telemetry.ExtractPeakValleys 提取）。
// 长度小于 3 时返回空结果（无法构成循环）。
func Count(peaks []float64) Result {
	res := Result{Residual: append([]float64(nil), peaks...)}
	if len(peaks) < 3 {
		return res
	}

	seq := make([]float64, len(peaks))
	copy(seq, peaks)

	// 四点法提取完整循环。
	for len(seq) >= 4 {
		extracted := false
		for i := 0; i+3 < len(seq); i++ {
			s1, s2, s3, s4 := seq[i], seq[i+1], seq[i+2], seq[i+3]
			d1 := math.Abs(s2 - s1)
			d2 := math.Abs(s3 - s2)
			d3 := math.Abs(s4 - s3)
			if d2 <= d1 && d2 <= d3 {
				res.Cycles = append(res.Cycles, Cycle{
					Amplitude: d2 / 2,
					Mean:      (s2 + s3) / 2,
					Count:     1.0,
				})
				res.FullCycles++
				// 移除 s2 与 s3（索引 i+1, i+2）。
				seq = append(seq[:i+1], seq[i+3:]...)
				extracted = true
				break
			}
		}
		if !extracted {
			break
		}
	}

	// 残差序列按腿数守恒配对：完整循环 + 半循环。
	res.Residual = append([]float64(nil), seq...)
	half, halfCount := halfCycles(seq)
	res.Cycles = append(res.Cycles, half...)
	res.HalfCycles = halfCount

	// 按幅值降序排序，便于下游损伤累计。
	sort.SliceStable(res.Cycles, func(i, j int) bool {
		return res.Cycles[i].Amplitude > res.Cycles[j].Amplitude
	})
	return res
}

// halfCycles 将残差峰谷序列配对为循环，返回 (循环列表, 半循环数)。
//
// 残差有 m 个点、m-1 条"腿"（相邻峰谷跨段）。每 2 条腿构成一个完整循环，
// 余 1 条腿构成一个半循环：
//   - 完整循环：相邻两两配对 (r[0],r[1]), (r[2],r[3]), ...；
//   - 半循环：最后一点与首点闭合（若余 1 条腿）。
func halfCycles(residual []float64) ([]Cycle, int) {
	m := len(residual)
	if m < 2 {
		return nil, 0
	}
	legs := m - 1
	full := legs / 2
	half := legs % 2
	out := make([]Cycle, 0, full+half)
	for i := 0; i < full; i++ {
		a, b := residual[2*i], residual[2*i+1]
		out = append(out, Cycle{
			Amplitude: math.Abs(a-b) / 2,
			Mean:      (a + b) / 2,
			Count:     1.0,
		})
	}
	if half > 0 {
		a, b := residual[m-1], residual[0]
		out = append(out, Cycle{
			Amplitude: math.Abs(a-b) / 2,
			Mean:      (a + b) / 2,
			Count:     0.5,
		})
	}
	return out, half
}

// TotalCycles 汇总循环计数（完整循环计 1，半循环计 0.5）。
func TotalCycles(cycles []Cycle) float64 {
	var sum float64
	for _, c := range cycles {
		sum += c.Count
	}
	return sum
}
