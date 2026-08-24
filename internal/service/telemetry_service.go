package service

import (
	"database/sql"

	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/store"
	"task224-bladecycle/internal/telemetry"
)

// IngestResult 是一次遥测接收的结果。
type IngestResult struct {
	SegmentID int64               `json:"segment_id"`
	Status    model.SegmentStatus `json:"status"`
	Inserted  int                 `json:"inserted"`
	Duplicate int                 `json:"duplicate"`
	Drift     bool                `json:"drift"`
	Gap       bool                `json:"gap"`
}

// TelemetryService 负责遥测接收、清洗、漂移标记与查询。
type TelemetryService struct {
	store *store.TelemetryStore
}

func NewTelemetryService(db *sql.DB) *TelemetryService {
	return &TelemetryService{store: store.NewTelemetryStore(db)}
}

// Ingest 接收一段遥测：
//
//  1. 校验输入（采样率一致、转速非负、序列非空且无 NaN/Inf）；
//  2. 峰谷提取（去除中间非极值点）；
//  3. 漂移检测（首尾均值偏移超过幅值 30% 判漂移）；
//  4. 缺口检测（同通道 seq_no 跳号判缺口）；
//  5. 幂等写入（UNIQUE(trial_id,channel_index,seq_no)）。
//
// 漂移/缺口段会被写入但状态标记为 drift/gap（保留原始数据，不参与计数）。
func (s *TelemetryService) Ingest(trial *model.Trial, channelIndex int, sensorID string, seqNo int64, rpm, temperature float64, strain []float64) (*IngestResult, error) {
	if !model.CanWriteTrial(trial.Status) {
		return nil, model.NewInvalidState("trial %d is sealed and immutable", trial.ID)
	}
	if err := telemetry.ValidateInput(trial.SampleRateHz, rpm, strain); err != nil {
		return nil, model.NewInvalidArgument("%v", err)
	}
	if _, err := s.store.EnsureChannel(trial.ID, channelIndex, sensorID); err != nil {
		return nil, err
	}

	peaks := telemetry.ExtractPeakValleys(strain)

	res := &IngestResult{}

	// 漂移检测（线性趋势判据，对称载荷不误判）。
	det := telemetry.NewDriftDetector()
	drift, _, _ := det.Detect(strain)
	res.Drift = drift

	// 缺口检测：同通道上一有效段序号。
	prev, err := s.lastSeqNo(trial.ID, channelIndex)
	if err != nil {
		return nil, err
	}
	if prev >= 0 && seqNo > prev+1 {
		res.Gap = true
	}

	status := model.SegmentValid
	reason := ""
	note := ""
	if drift {
		status = model.SegmentDrift
		reason = "mean shift detected"
	}
	if res.Gap {
		status = model.SegmentGap
		note = "seq_no gap detected"
	}
	if drift && res.Gap {
		// 漂移优先（缺口说明保留在 note）。
		note = "seq_no gap detected"
	}

	seg := &model.TelemetrySegment{
		TrialID:      trial.ID,
		ChannelIndex: channelIndex,
		SeqNo:        seqNo,
		RPM:          rpm,
		Temperature:  temperature,
		StrainPeaks:  peaks,
		Status:       status,
		DriftReason:  reason,
		GapNote:      note,
	}
	id, err := s.store.InsertSegment(seg)
	if err == model.ErrDuplicate {
		res.Duplicate = 1
		res.Status = model.SegmentDuplicate
		return res, nil
	}
	if err != nil {
		return nil, err
	}
	res.SegmentID = id
	res.Status = status
	res.Inserted = 1
	return res, nil
}

// MarkDrift 将某段标记为传感器漂移（保留数据，不参与计数）。
func (s *TelemetryService) MarkDrift(segmentID int64, reason string) error {
	seg, err := s.store.GetSegment(segmentID)
	if err == sql.ErrNoRows {
		return model.NewNotFound("segment %d not found", segmentID)
	}
	if err != nil {
		return err
	}
	if seg.Status == model.SegmentDuplicate {
		return model.NewInvalidState("duplicate segment cannot be marked")
	}
	return s.store.UpdateStatus(segmentID, model.SegmentDrift, reason, seg.GapNote)
}

// ListByTrial 列出某试验全部遥测段。
func (s *TelemetryService) ListByTrial(trialID int64) ([]*model.TelemetrySegment, error) {
	return s.store.ListByTrial(trialID)
}

// CountDrift 统计某试验漂移段数量。
func (s *TelemetryService) CountDrift(trialID int64) (int, error) {
	return s.store.CountByStatus(trialID, model.SegmentDrift)
}

// lastSeqNo 返回同通道最大序号，无记录时返回 -1。
func (s *TelemetryService) lastSeqNo(trialID int64, channelIndex int) (int64, error) {
	var maxSeq sql.NullInt64
	if err := s.store.MaxSeq(trialID, channelIndex, &maxSeq); err != nil {
		return 0, err
	}
	if !maxSeq.Valid {
		return -1, nil
	}
	return maxSeq.Int64, nil
}
