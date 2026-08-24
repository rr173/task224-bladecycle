package rainflow

import (
	"math"
	"testing"
)

func TestCountSingleCycle(t *testing.T) {
	// 峰谷序列 [0, 2, 0, 2, 0] 应提取一个完整循环（幅值 1）+ 残差半循环。
	res := Count([]float64{0, 2, 0, 2, 0})
	if res.FullCycles < 1 {
		t.Fatalf("expected at least 1 full cycle, got %d", res.FullCycles)
	}
	// 总循环计数守恒：4 个半循环跨度 = 2.0 循环。
	total := TotalCycles(res.Cycles)
	if math.Abs(total-2.0) > 1e-9 {
		t.Fatalf("expected total cycles 2.0, got %f", total)
	}
}

func TestCountAmplitude(t *testing.T) {
	// 幅值 = 半幅 = |峰-谷|/2。
	res := Count([]float64{0, 10, 0, 10, 0})
	for _, c := range res.Cycles {
		if math.Abs(c.Amplitude-5.0) > 1e-9 {
			t.Fatalf("expected amplitude 5.0, got %f", c.Amplitude)
		}
	}
}

func TestCountEmpty(t *testing.T) {
	res := Count(nil)
	if len(res.Cycles) != 0 {
		t.Fatalf("expected no cycles for empty input")
	}
	res = Count([]float64{1})
	if len(res.Cycles) != 0 {
		t.Fatalf("expected no cycles for single point")
	}
}

func TestCountNested(t *testing.T) {
	// 嵌套载荷：大循环内嵌小循环，四点法应正确提取。
	// 峰谷序列：0 → 5(大峰) → 2 → 3(小峰) → 1(小谷) → 5 → 0
	seq := []float64{0, 5, 2, 3, 1, 5, 0}
	res := Count(seq)
	if res.FullCycles == 0 {
		t.Fatalf("expected nested cycles to be extracted, got full=%d", res.FullCycles)
	}
	for _, c := range res.Cycles {
		if c.Count != 1.0 && c.Count != 0.5 {
			t.Fatalf("invalid cycle count %f", c.Count)
		}
	}
}

func TestTotalCycles(t *testing.T) {
	cycles := []Cycle{
		{Amplitude: 5, Count: 1.0},
		{Amplitude: 3, Count: 0.5},
		{Amplitude: 1, Count: 0.5},
	}
	if got := TotalCycles(cycles); got != 2.0 {
		t.Fatalf("expected 2.0, got %f", got)
	}
}
