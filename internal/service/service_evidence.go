package service

import (
	"fmt"

	"task215-fiberlineage/internal/evidence"
	"task215-fiberlineage/internal/model"
)

// AddEvidence 在两个样本间新增证据项（pending），并刷新它们之间的相似度边。
func (svc *Service) AddEvidence(sampleA, sampleB int64, kind model.EvidenceKind, weight float64, note string) (*model.Evidence, error) {
	ev := &model.Evidence{
		SampleA: sampleA, SampleB: sampleB,
		Kind: kind, Weight: weight, Note: note,
	}
	if err := svc.store.CreateEvidence(ev); err != nil {
		return nil, err
	}
	if err := svc.recomputePair(sampleA, sampleB); err != nil {
		return ev, fmt.Errorf("evidence added but similarity recompute failed: %w", err)
	}
	return ev, nil
}

// ListEvidence 列出证据（可按 kind/status 过滤）。
func (svc *Service) ListEvidence(kind, status string) ([]model.Evidence, error) {
	return svc.store.ListEvidence(kind, status)
}

// UpdateEvidence 更新证据权重/备注（不限状态）。
func (svc *Service) UpdateEvidence(id int64, weight float64, note string) (*model.Evidence, error) {
	return svc.store.UpdateEvidence(id, weight, "", note)
}

// ValidateEvidence 将证据状态从校验中推进到有效/冲突/排除，并校验状态机流转合法性。
func (svc *Service) ValidateEvidence(id int64, to model.EvidenceStatus) (*model.Evidence, error) {
	cur, err := svc.store.GetEvidence(id)
	if err != nil {
		return nil, err
	}
	if !evidence.CanTransition(cur.Status, to) {
		return nil, fmt.Errorf("%w: evidence %s -> %s", model.ErrInvalidState, cur.Status, to)
	}
	updated, err := svc.store.UpdateEvidence(id, -1, string(to), "")
	if err != nil {
		return nil, err
	}
	// 待核验证据不计入相似度（留给研究者校验），其状态变化不触发重算，
	// 避免无谓写入并保持「待核验 -> 相似度为零」的合理边界。
	// 有效证据则会贡献权重，必须立即刷新样本对相似度，避免残留过期的零分边。
	if to == model.EvidencePending {
		return updated, nil
	}
	if err := svc.recomputePair(cur.SampleA, cur.SampleB); err != nil {
		return updated, fmt.Errorf("evidence validated but similarity recompute failed: %w", err)
	}
	return updated, nil
}

// ExcludeEvidence 排除证据（等价于 ValidateEvidence 到 excluded），并清理相似度贡献。
func (svc *Service) ExcludeEvidence(id int64) (*model.Evidence, error) {
	return svc.ValidateEvidence(id, model.EvidenceExcluded)
}
