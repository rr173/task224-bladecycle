package telemetry

import (
	"math"
	"testing"
)

func TestExtractPeakValleys(t *testing.T) {
	// 单调上升序列：无中间极值，仅保留端点。
	got := ExtractPeakValleys([]float64{1, 2, 3, 4, 5})
	if len(got) != 2 || got[0] != 1 || got[1] != 5 {
		t.Fatalf("monotonic series should collapse to endpoints, got %v", got)
	}
}

func TestExtractPeakValleysZigzag(t *testing.T) {
	// 锯齿序列：每个内部点都是极值。
	got := ExtractPeakValleys([]float64{0, 10, 0, 10, 0})
	if len(got) != 5 {
		t.Fatalf("zigzag should keep all points, got %v", got)
	}
}

func TestExtractPeakValleysPlateau(t *testing.T) {
	// 平台（连续相等）应被合并。
	got := ExtractPeakValleys([]float64{0, 5, 5, 5, 0})
	if len(got) != 3 {
		t.Fatalf("plateau should merge equal points, got %v", got)
	}
}

func TestDetectDriftRamp(t *testing.T) {
	det := NewDriftDetector()
	// 纯线性斜坡应被判定为漂移。
	n := 100
	series := make([]float64, n)
	for i := 0; i < n; i++ {
		series[i] = 10.0 * float64(i) / float64(n)
	}
	drift, _, _ := det.Detect(series)
	if !drift {
		t.Fatalf("linear ramp should be detected as drift")
	}
}

func TestDetectDriftSinusoid(t *testing.T) {
	det := NewDriftDetector()
	// 对称正弦载荷不应被判定为漂移。
	n := 1000
	series := make([]float64, n)
	for i := 0; i < n; i++ {
		tval := float64(i) / 1000.0
		series[i] = 120.0 * math.Sin(2*math.Pi*2*tval)
	}
	drift, _, _ := det.Detect(series)
	if drift {
		t.Fatalf("symmetric sinusoid should not be detected as drift")
	}
}

func TestValidateInput(t *testing.T) {
	if err := ValidateInput(0, 0, []float64{1}); err == nil {
		t.Fatalf("expected error for non-positive sample rate")
	}
	if err := ValidateInput(1000, 0, []float64{1, math.NaN()}); err == nil {
		t.Fatalf("expected error for NaN in series")
	}
	if err := ValidateInput(1000, 12000, []float64{1, 2, 3}); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
}
