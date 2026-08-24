package telemetry

import "math"

// DriftDetector 检测应变序列中的传感器漂移。
//
// 漂移判据：对序列做最小二乘线性拟合，若线性趋势总量（|斜率| × 跨度）
// 超过序列幅值范围的一定比例（默认 50%），且拟合相关系数足够显著
// （默认 |r| ≥ 0.9），则判定为漂移。对称载荷（正弦等）线性趋势趋近 0，
// 不会被误判；单调斜坡载荷则显著命中。
type DriftDetector struct {
	TrendRatio float64 // 趋势总量 / 幅值范围 的阈值（默认 0.5）
	MinR       float64 // 线性拟合最小 |相关系数|（默认 0.9）
}

// NewDriftDetector 构造默认漂移检测器。
func NewDriftDetector() *DriftDetector {
	return &DriftDetector{
		TrendRatio: 0.5,
		MinR:       0.9,
	}
}

// Detect 返回 (是否漂移, 斜率, 相关系数)。
func (d *DriftDetector) Detect(series []float64) (bool, float64, float64) {
	n := len(series)
	if n < 4 {
		return false, 0, 0
	}
	slope, r := linearFit(series)
	totalTrend := math.Abs(slope) * float64(n-1)
	ampRange := Range(series)

	drift := false
	if ampRange > 0 && totalTrend > ampRange*d.TrendRatio && math.Abs(r) >= d.MinR {
		drift = true
	}
	return drift, slope, r
}

// linearFit 对序列做最小二乘线性拟合，返回斜率与皮尔逊相关系数。
func linearFit(y []float64) (slope float64, r float64) {
	n := float64(len(y))
	var sx, sy, sxy, sxx, syy float64
	for i, v := range y {
		x := float64(i)
		sx += x
		sy += v
		sxy += x * v
		sxx += x * x
		syy += v * v
	}
	denom := n*sxx - sx*sx
	if denom == 0 {
		return 0, 0
	}
	slope = (n*sxy - sx*sy) / denom
	num := n*sxy - sx*sy
	denR := math.Sqrt((n*sxx - sx*sx) * (n*syy - sy*sy))
	if denR == 0 {
		return slope, 0
	}
	r = num / denR
	return slope, r
}
