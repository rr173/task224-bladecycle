package service

import (
	"database/sql"

	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/store"
)

// TrialService 管理试验架次生命周期。
type TrialService struct {
	store *store.TrialStore
	mats  *store.MaterialStore
}

func NewTrialService(db *sql.DB) *TrialService {
	return &TrialService{
		store: store.NewTrialStore(db),
		mats:  store.NewMaterialStore(db),
	}
}

// Create 创建试验架次（初始 ready）。
//
// 在落库前校验材料批次存在，避免留下无有效材料引用的试验记录：
// 不存在的批次号在创建阶段即被拒绝，数据库中不写入该条记录。
func (s *TrialService) Create(name, engineModel string, materialBatchID int64, sampleRateHz float64) (*model.Trial, error) {
	if name == "" {
		return nil, model.NewInvalidArgument("trial name must not be empty")
	}
	if engineModel == "" {
		return nil, model.NewInvalidArgument("engine model must not be empty")
	}
	if sampleRateHz <= 0 {
		return nil, model.NewInvalidArgument("sample rate must be positive")
	}
	if _, err := s.mats.Get(materialBatchID); err == sql.ErrNoRows {
		return nil, model.NewNotFound("material batch %d not found", materialBatchID)
	} else if err != nil {
		return nil, err
	}
	id, err := s.store.Insert(name, engineModel, materialBatchID, sampleRateHz)
	if err != nil {
		return nil, err
	}
	return s.store.Get(id)
}

// Get 按 ID 查询试验。
func (s *TrialService) Get(id int64) (*model.Trial, error) {
	t, err := s.store.Get(id)
	if err == sql.ErrNoRows {
		return nil, model.NewNotFound("trial %d not found", id)
	}
	return t, err
}

// List 列出全部试验。
func (s *TrialService) List() ([]*model.Trial, error) {
	return s.store.List()
}

// Start 将试验从 ready 推进到 running（开始接收遥测）。
func (s *TrialService) Start(id int64) (*model.Trial, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := model.Transition(t.Status, model.TrialRunning); err != nil {
		return nil, err
	}
	if err := s.store.UpdateStatus(id, model.TrialRunning); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// FinishAcquisition 将试验从 running 推进到 analyzing（停止接收，进入分析）。
func (s *TrialService) FinishAcquisition(id int64) (*model.Trial, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := model.Transition(t.Status, model.TrialAnalyzing); err != nil {
		return nil, err
	}
	if err := s.store.UpdateStatus(id, model.TrialAnalyzing); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Confirm 将试验从 analyzing 推进到 confirmed（损伤结论已复核）。
func (s *TrialService) Confirm(id int64) (*model.Trial, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := model.Transition(t.Status, model.TrialConfirmed); err != nil {
		return nil, err
	}
	if err := s.store.UpdateStatus(id, model.TrialConfirmed); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// Seal 将试验从 confirmed 推进到 sealed（封存，单向终态）。
func (s *TrialService) Seal(id int64) (*model.Trial, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := model.Transition(t.Status, model.TrialSealed); err != nil {
		return nil, err
	}
	if err := s.store.UpdateStatus(id, model.TrialSealed); err != nil {
		return nil, err
	}
	return s.Get(id)
}

// EnsureWritable 校验试验未封存，可继续写入。
func (s *TrialService) EnsureWritable(id int64) (*model.Trial, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if !model.CanWriteTrial(t.Status) {
		return nil, model.NewInvalidState("trial %d is sealed and immutable", id)
	}
	return t, nil
}
