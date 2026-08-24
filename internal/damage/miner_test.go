package damage

import (
	"testing"

	"task224-bladecycle/internal/material"
	"task224-bladecycle/internal/rainflow"
)

func TestAccumulateZeroDamage(t *testing.T) {
	c := material.SNCurve{FatigueStrengthCoef: 200, FatigueExponent: -0.12, UltimateStrength: 300, MeanStressMethod: "goodman"}
	// 幅值极低的循环不应贡献损伤。
	cycles := []rainflow.Cycle{
		{Amplitude: 1e-6, Mean: 0, Count: 1000},
	}
	dmg, total := Accumulate(cycles, c)
	if dmg < 0 {
		t.Fatalf("damage should be non-negative, got %f", dmg)
	}
	if total != 1000 {
		t.Fatalf("total cycles should be 1000, got %f", total)
	}
}

func TestAccumulateMonotonic(t *testing.T) {
	c := material.SNCurve{FatigueStrengthCoef: 200, FatigueExponent: -0.12, UltimateStrength: 300, MeanStressMethod: "goodman"}
	// 高幅值循环损伤应大于低幅值。
	high := []rainflow.Cycle{{Amplitude: 150, Mean: 0, Count: 1}}
	low := []rainflow.Cycle{{Amplitude: 50, Mean: 0, Count: 1}}
	dHigh, _ := Accumulate(high, c)
	dLow, _ := Accumulate(low, c)
	if dHigh <= dLow {
		t.Fatalf("higher amplitude should cause more damage: %f vs %f", dHigh, dLow)
	}
}

func TestRemainingLife(t *testing.T) {
	if got := RemainingLife(0.5); got != 50.0 {
		t.Fatalf("expected 50%%, got %f", got)
	}
	if got := RemainingLife(1.5); got != 0 {
		t.Fatalf("damage over threshold should yield 0%%, got %f", got)
	}
}

func TestBinDamage(t *testing.T) {
	c := material.SNCurve{FatigueStrengthCoef: 200, FatigueExponent: -0.12, UltimateStrength: 300, MeanStressMethod: "goodman"}
	cycles := []rainflow.Cycle{
		{Amplitude: 150, Mean: 0, Count: 1},
		{Amplitude: 50, Mean: 0, Count: 1},
	}
	bins := BinDamage(cycles, c, []float64{0, 100, 200})
	if len(bins) != 2 {
		t.Fatalf("expected 2 bins, got %d", len(bins))
	}
	if bins[1].CycleCount != 1 {
		t.Fatalf("expected 1 cycle in upper bin, got %f", bins[1].CycleCount)
	}
	if bins[0].CycleCount != 1 {
		t.Fatalf("expected 1 cycle in lower bin, got %f", bins[0].CycleCount)
	}
}
