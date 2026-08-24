package service

import (
	"database/sql"

	"task224-bladecycle/internal/material"
	"task224-bladecycle/internal/model"
	"task224-bladecycle/internal/store"
)

// MaterialService 管理材料批次（S-N 曲线参数）。
type MaterialService struct {
	store *store.MaterialStore
}

func NewMaterialService(db *sql.DB) *MaterialService {
	return &MaterialService{store: store.NewMaterialStore(db)}
}

// Create 校验并创建材料批次。
func (s *MaterialService) Create(code, name string, strengthCoef, exponent, ultimate float64, method string) (*model.Material, error) {
	m := &model.Material{
		Code:                code,
		Name:                name,
		FatigueStrengthCoef: strengthCoef,
		FatigueExponent:     exponent,
		UltimateStrength:    ultimate,
		MeanStressMethod:    method,
	}
	curve := material.SNCurve{
		FatigueStrengthCoef: strengthCoef,
		FatigueExponent:     exponent,
		UltimateStrength:    ultimate,
		MeanStressMethod:    method,
	}
	if err := curve.Validate(); err != nil {
		return nil, model.NewInvalidArgument("%v", err)
	}
	if code == "" {
		return nil, model.NewInvalidArgument("material code must not be empty")
	}
	id, err := s.store.Insert(m)
	if err != nil {
		return nil, model.NewConflict("material code %q already exists or insert failed: %v", code, err)
	}
	m.ID = id
	return m, nil
}

// Get 按 ID 查询材料。
func (s *MaterialService) Get(id int64) (*model.Material, error) {
	m, err := s.store.Get(id)
	if err == sql.ErrNoRows {
		return nil, model.NewNotFound("material %d not found", id)
	}
	return m, err
}

// List 列出全部材料。
func (s *MaterialService) List() ([]*model.Material, error) {
	return s.store.List()
}
