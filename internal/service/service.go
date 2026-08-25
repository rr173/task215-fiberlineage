// Package service 是业务编排层：组合 store 与各纯逻辑业务包，
// 在校验状态机与不变量的前提下暴露高层操作。httpapi 仅调用本层，不直接碰 store。
package service

import (
	"fmt"

	"task215-fiberlineage/internal/model"
	"task215-fiberlineage/internal/spectrum"
	"task215-fiberlineage/internal/store"
)

// Service 持有持久化与业务规则，是 API 与领域之间的唯一编排入口。
type Service struct {
	store *store.Store
}

// New 构建 Service。
func New(s *store.Store) *Service { return &Service{store: s} }

// Store 暴露底层 store（仅供自检与事务场景）。
func (svc *Service) Store() *store.Store { return svc.store }

// ---- 样本 ----

// RegisterSample 登记文献样本（状态 registered）。
func (svc *Service) RegisterSample(code, title, source string) (*model.Sample, error) {
	if code == "" || title == "" || source == "" {
		return nil, fmt.Errorf("%w: code/title/source required", model.ErrInvalidInput)
	}
	sm := &model.Sample{Code: code, Title: title, Source: source, Status: model.SampleRegistered}
	if err := svc.store.CreateSample(sm); err != nil {
		return nil, err
	}
	return sm, nil
}

// GetSample 获取样本。
func (svc *Service) GetSample(id int64) (*model.Sample, error) { return svc.store.GetSample(id) }

// ListSamples 列出样本。
func (svc *Service) ListSamples() ([]model.Sample, error) { return svc.store.ListSamples() }

// UpdateSample 更新样本基础信息。
func (svc *Service) UpdateSample(id int64, title, source, status string) (*model.Sample, error) {
	return svc.store.UpdateSample(id, title, source, status)
}

// SealSample 封存样本。
func (svc *Service) SealSample(id int64) (*model.Sample, error) { return svc.store.SealSample(id) }

// SubmitDetection 提交样本检测谱：先经 spectrum 校验，再持久化并推进状态。
func (svc *Service) SubmitDetection(sampleID int64, det *model.SampleDetection) error {
	if det == nil {
		return fmt.Errorf("%w: nil detection", model.ErrInvalidInput)
	}
	det.SampleID = sampleID
	if err := spectrum.Validate(det); err != nil {
		return err
	}
	norm := spectrum.NormalizeIntensities(*det)
	det.FiberSpectrum = norm.FiberSpectrum
	det.DyePeaks = norm.DyePeaks
	det.RepairLayers = norm.RepairLayers
	return svc.store.SetDetection(sampleID, det)
}

// GetDetection 读取样本检测谱。
func (svc *Service) GetDetection(sampleID int64) (*model.SampleDetection, error) {
	return svc.store.GetDetection(sampleID)
}
