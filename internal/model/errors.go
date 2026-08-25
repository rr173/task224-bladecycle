package model

import (
	"errors"
	"fmt"
)

// 领域错误，HTTP 层据此映射状态码。
var (
	ErrNotFound         = errors.New("resource not found")
	ErrConflict         = errors.New("resource conflict")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrInvalidState     = errors.New("invalid state transition")
	ErrSealed           = errors.New("trial is sealed and immutable")
	ErrDuplicate        = errors.New("duplicate telemetry segment")
)

// DomainError 携带错误码与可读信息。
type DomainError struct {
	Kind error
	Msg  string
}

func (e *DomainError) Error() string {
	return e.Msg
}

func (e *DomainError) Unwrap() error {
	return e.Kind
}

// NewNotFound 构造 NotFound 错误。
func NewNotFound(format string, args ...interface{}) error {
	return &DomainError{Kind: ErrNotFound, Msg: fmt.Sprintf(format, args...)}
}

// NewConflict 构造 Conflict 错误。
func NewConflict(format string, args ...interface{}) error {
	return &DomainError{Kind: ErrConflict, Msg: fmt.Sprintf(format, args...)}
}

// NewInvalidArgument 构造参数错误。
func NewInvalidArgument(format string, args ...interface{}) error {
	return &DomainError{Kind: ErrInvalidArgument, Msg: fmt.Sprintf(format, args...)}
}

// NewInvalidState 构造状态机流转错误。
func NewInvalidState(format string, args ...interface{}) error {
	return &DomainError{Kind: ErrInvalidState, Msg: fmt.Sprintf(format, args...)}
}

// IsNotFound 判断是否 NotFound。
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsConflict 判断是否 Conflict。
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsInvalidArgument 判断是否参数错误。
func IsInvalidArgument(err error) bool {
	return errors.Is(err, ErrInvalidArgument)
}

// IsInvalidState 判断是否状态机错误。
func IsInvalidState(err error) bool {
	return errors.Is(err, ErrInvalidState)
}

// Transition 校验并返回 trial 状态机流转是否合法，返回目标状态。
func Transition(from TrialStatus, to TrialStatus) error {
	allowed := map[TrialStatus]map[TrialStatus]bool{
		TrialReady:     {TrialRunning: true},
		TrialRunning:   {TrialAnalyzing: true},
		TrialAnalyzing: {TrialConfirmed: true},
		TrialConfirmed: {TrialSealed: true},
		TrialSealed:    {},
	}
	if from == to {
		return nil
	}
	if targets, ok := allowed[from]; ok && targets[to] {
		return nil
	}
	return NewInvalidState("trial cannot transition from %s to %s", from, to)
}

// CanAcceptTelemetry 判断 trial 是否仍处于采集阶段，可接收新遥测。
// 仅 running 态可写入；ready 尚未开始、analyzing 及之后采集已结束。
func CanAcceptTelemetry(s TrialStatus) bool {
	return s == TrialRunning
}

// CanWriteTrial 判断 trial 是否仍可写入（向后兼容别名，仅运行中可写入）。
func CanWriteTrial(s TrialStatus) bool {
	return CanAcceptTelemetry(s)
}

// ValidCycleTransition 判断 cycle 状态机流转合法性。
func ValidCycleTransition(from CycleStatus, to CycleStatus) bool {
	allowed := map[CycleStatus][]CycleStatus{
		CycleCandidate: {CycleCounted, CycleReview, CycleConfirmed},
		CycleCounted:   {CycleReview, CycleConfirmed},
		CycleReview:    {CycleConfirmed},
		CycleConfirmed: {},
	}
	for _, t := range allowed[from] {
		if t == to {
			return true
		}
	}
	return from == to
}
