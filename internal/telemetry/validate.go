package telemetry

import (
	"fmt"
)

// ValidateInput 校验一段遥测输入的合法性，返回校验错误（nil 表示通过）。
//
// 规则：
//   - 采样率必须为正；
//   - 转速非负；
//   - 应变序列非空；
//   - 序列内不存在 NaN / Inf。
func ValidateInput(sampleRateHz, rpm float64, strain []float64) error {
	if sampleRateHz <= 0 {
		return fmt.Errorf("sample rate must be positive, got %f", sampleRateHz)
	}
	if rpm < 0 {
		return fmt.Errorf("rpm must be non-negative, got %f", rpm)
	}
	if len(strain) == 0 {
		return fmt.Errorf("strain series must not be empty")
	}
	for i, x := range strain {
		if x != x || x > 1e12 || x < -1e12 {
			return fmt.Errorf("strain series contains invalid value at index %d", i)
		}
	}
	return nil
}

// ValidateSeqNo 校验段序号单调性：seqNo 必须大于前一序号。
func ValidateSeqNo(prevSeqNo, seqNo int64) error {
	if seqNo <= prevSeqNo {
		return fmt.Errorf("seq_no must be strictly increasing: prev=%d got=%d", prevSeqNo, seqNo)
	}
	return nil
}

// ValidateRateConsistency 校验本次采样率与试验登记采样率一致。
func ValidateRateConsistency(trialRate, segRate float64) error {
	if trialRate != segRate {
		return fmt.Errorf("sample rate mismatch: trial=%f segment=%f", trialRate, segRate)
	}
	return nil
}
