// Package telemetry 负责遥测数据的清洗：峰谷提取、采样率/序号/单调性校验、
// 传感器漂移检测。清洗后的峰谷序列交给 rainflow 包做循环计数。
package telemetry

import "math"

// ExtractPeakValleys 从原始应变序列提取峰谷序列（去掉中间非极值点）。
//
// 处理规则：
//   - 连续相等点合并，避免伪峰谷；
//   - 严格局部极值：上升后下降为峰，下降后上升为谷；
//   - 首尾点保留为半循环端点。
//
// 输入序列长度小于 2 时原样返回。
func ExtractPeakValleys(series []float64) []float64 {
	if len(series) < 2 {
		return series
	}
	// 先压缩相邻等值点。
	compressed := make([]float64, 0, len(series))
	compressed = append(compressed, series[0])
	for i := 1; i < len(series); i++ {
		if series[i] != compressed[len(compressed)-1] {
			compressed = append(compressed, series[i])
		}
	}
	if len(compressed) < 3 {
		return compressed
	}

	out := make([]float64, 0, len(compressed))
	out = append(out, compressed[0])
	for i := 1; i < len(compressed)-1; i++ {
		prev := compressed[i-1]
		cur := compressed[i]
		next := compressed[i+1]
		isPeak := cur > prev && cur >= next
		isValley := cur < prev && cur <= next
		if isPeak || isValley {
			out = append(out, cur)
		}
	}
	out = append(out, compressed[len(compressed)-1])
	return out
}

// DetectGap 检测序列中是否存在缺口：相邻点间距异常（按采样率推算的时间戳
// 缺失）。通过相邻序号差值判断，gapThreshold 为允许的最大序号跨度。
func DetectGap(seqNos []int64, gapThreshold int64) (bool, int64) {
	for i := 1; i < len(seqNos); i++ {
		delta := seqNos[i] - seqNos[i-1]
		if delta > gapThreshold {
			return true, seqNos[i-1]
		}
	}
	return false, 0
}

// Mean 计算序列均值。
func Mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var s float64
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

// StdDev 计算序列标准差（总体）。
func StdDev(v []float64) float64 {
	if len(v) < 2 {
		return 0
	}
	m := Mean(v)
	var s float64
	for _, x := range v {
		d := x - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(v)))
}

// Range 返回序列幅值范围（max - min）。
func Range(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	min, max := v[0], v[0]
	for _, x := range v {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	return max - min
}
