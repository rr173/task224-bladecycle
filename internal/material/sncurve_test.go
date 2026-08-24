package material

import (
	"math"
	"testing"
)

func TestLifeCycles(t *testing.T) {
	c := SNCurve{FatigueStrengthCoef: 200, FatigueExponent: -0.12, UltimateStrength: 300, MeanStressMethod: "goodman"}
	// 应力幅等于疲劳强度系数时，2N_f = 1，N_f = 0.5。
	nf := c.LifeCycles(200)
	if math.Abs(nf-0.5) > 1e-9 {
		t.Fatalf("expected N_f=0.5 at amplitude=coef, got %f", nf)
	}
	// 应力幅为 0 时寿命无穷。
	if !math.IsInf(c.LifeCycles(0), 1) {
		t.Fatalf("expected infinite life at zero amplitude")
	}
	// 应力幅减半，寿命应增大（b 为负）。
	nfLow := c.LifeCycles(100)
	if nfLow <= nf {
		t.Fatalf("lower amplitude should yield longer life: %f vs %f", nfLow, nf)
	}
}

func TestCorrectAmplitudeGoodman(t *testing.T) {
	c := SNCurve{FatigueStrengthCoef: 200, FatigueExponent: -0.12, UltimateStrength: 300, MeanStressMethod: "goodman"}
	// 平均应力 0 时不变。
	if got := c.CorrectAmplitude(100, 0); got != 100 {
		t.Fatalf("zero mean should not change amplitude, got %f", got)
	}
	// 拉伸平均应力 150，Goodman：σ_ar = 100 / (1 - 150/300) = 200。
	got := c.CorrectAmplitude(100, 150)
	if math.Abs(got-200.0) > 1e-9 {
		t.Fatalf("goodman correction expected 200, got %f", got)
	}
}

func TestCorrectAmplitudeMorrow(t *testing.T) {
	c := SNCurve{FatigueStrengthCoef: 200, FatigueExponent: -0.12, UltimateStrength: 300, MeanStressMethod: "morrow"}
	// 拉伸平均应力 100，Morrow：σ_ar = 100 / (1 - 100/200) = 200。
	got := c.CorrectAmplitude(100, 100)
	if math.Abs(got-200.0) > 1e-9 {
		t.Fatalf("morrow correction expected 200, got %f", got)
	}
}

func TestValidate(t *testing.T) {
	c := SNCurve{FatigueStrengthCoef: 200, FatigueExponent: -0.12, UltimateStrength: 300, MeanStressMethod: "goodman"}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid curve rejected: %v", err)
	}
	bad := SNCurve{FatigueStrengthCoef: 200, FatigueExponent: 0.12, UltimateStrength: 300, MeanStressMethod: "goodman"}
	if err := bad.Validate(); err == nil {
		t.Fatalf("positive exponent should be rejected")
	}
}
