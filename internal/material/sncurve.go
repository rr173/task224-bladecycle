// Package material 封装材料疲劳参数：Basquin S-N 曲线寿命计算与平均应力
// 修正（Goodman / Morrow）。供 damage 包在 Miner 累计时调用。
package material

import "math"

// SNCurve 是材料的 Basquin 疲劳参数。
//
//	σ_a = σ_f' · (2N_f)^b
//	N_f  = 0.5 · (σ_a / σ_f')^(1/b)
type SNCurve struct {
	FatigueStrengthCoef float64 // σ_f'（疲劳强度系数，MPa）
	FatigueExponent     float64 // b（疲劳强度指数，通常为负）
	UltimateStrength    float64 // σ_u（极限抗拉强度，MPa，Goodman 用）
	MeanStressMethod    string  // goodman | morrow
}

// LifeCycles 计算给定应力幅（半幅，MPa）下的疲劳寿命 N_f。
// 返回循环数（完整循环），半循环需按 0.5 折算由调用方处理。
func (c SNCurve) LifeCycles(stressAmplitude float64) float64 {
	if stressAmplitude <= 0 {
		return math.Inf(1)
	}
	if c.FatigueStrengthCoef <= 0 || c.FatigueExponent == 0 {
		return math.Inf(1)
	}
	ratio := stressAmplitude / c.FatigueStrengthCoef
	// ratio 可能 <=0 或 >1，直接 pow。
	return 0.5 * math.Pow(ratio, 1.0/c.FatigueExponent)
}

// CorrectAmplitude 对平均应力做修正，返回等效完全反向应力幅 σ_ar。
//
//   - Goodman: σ_ar = σ_a / (1 - σ_m / σ_u)
//   - Morrow:  σ_ar = σ_a / (1 - σ_m / σ_f')
//
// 当平均应力为 0 时直接返回原幅值；分母趋近 0 或为负时返回 +Inf（无法承载）。
func (c SNCurve) CorrectAmplitude(stressAmplitude, meanStress float64) float64 {
	if meanStress == 0 {
		return stressAmplitude
	}
	var denom float64
	switch c.MeanStressMethod {
	case "morrow":
		if c.FatigueStrengthCoef == 0 {
			return math.Inf(1)
		}
		denom = 1 - meanStress/c.FatigueStrengthCoef
	default: // goodman
		if c.UltimateStrength == 0 {
			return math.Inf(1)
		}
		denom = 1 - meanStress/c.UltimateStrength
	}
	if denom <= 0 {
		return math.Inf(1)
	}
	return stressAmplitude / denom
}

// Validate 校验材料参数是否合法（系数正、指数负、极限强度正）。
func (c SNCurve) Validate() error {
	if c.FatigueStrengthCoef <= 0 {
		return errInvalid("fatigue strength coefficient must be positive")
	}
	if c.FatigueExponent >= 0 {
		return errInvalid("fatigue exponent must be negative")
	}
	if c.UltimateStrength <= 0 {
		return errInvalid("ultimate strength must be positive")
	}
	if c.MeanStressMethod != "goodman" && c.MeanStressMethod != "morrow" {
		return errInvalid("mean stress method must be goodman or morrow")
	}
	return nil
}

type materialError struct{ msg string }

func (e *materialError) Error() string { return e.msg }

func errInvalid(msg string) error { return &materialError{msg: msg} }
