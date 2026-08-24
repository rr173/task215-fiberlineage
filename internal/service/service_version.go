package service

import (
	"fmt"
	"time"

	"task215-fiberlineage/internal/model"
	"task215-fiberlineage/internal/version"
)

// CreateVersion 创建研究版本（编辑中）。baselineID 为差异基准版本（0 表示无）。
func (svc *Service) CreateVersion(code string, baselineID int64) (*model.ResearchVersion, error) {
	if code == "" {
		return nil, fmt.Errorf("%w: code required", model.ErrInvalidInput)
	}
	v := &model.ResearchVersion{Code: code, Status: model.VersionEditing, BaselineVersionID: baselineID}
	if err := svc.store.CreateResearchVersion(v); err != nil {
		return nil, err
	}
	if baselineID > 0 {
		if err := svc.store.SetVersionBaseline(v.ID, baselineID); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// ListVersions 列出版本。
func (svc *Service) ListVersions() ([]model.ResearchVersion, error) { return svc.store.ListResearchVersions() }

// GetVersion 获取版本。
func (svc *Service) GetVersion(id int64) (*model.ResearchVersion, error) {
	return svc.store.GetResearchVersion(id)
}

// FreezeVersion 冻结版本（editing -> frozen）。要求版本至少含一个已确认假设，冻结后保留全部证据不可变。
func (svc *Service) FreezeVersion(id int64) (*model.ResearchVersion, error) {
	v, err := svc.store.GetResearchVersion(id)
	if err != nil {
		return nil, err
	}
	if v.Status != model.VersionEditing {
		return nil, fmt.Errorf("%w: only editing version can be frozen (got %s)", model.ErrInvalidState, v.Status)
	}
	hyps, err := svc.store.ListLineageHypotheses(id)
	if err != nil {
		return nil, err
	}
	hasConfirmed := false
	for _, h := range hyps {
		if h.Status == model.LineageConfirmed {
			hasConfirmed = true
			break
		}
	}
	if !hasConfirmed {
		return nil, fmt.Errorf("%w: version needs at least one confirmed hypothesis to freeze", model.ErrInvalidInput)
	}
	now := time.Now().UTC()
	return svc.store.UpdateResearchVersion(id, string(model.VersionFrozen), &now)
}

// PublishVersion 共享版本（editing -> shared）。
func (svc *Service) PublishVersion(id int64) (*model.ResearchVersion, error) {
	v, err := svc.store.GetResearchVersion(id)
	if err != nil {
		return nil, err
	}
	if v.Status != model.VersionEditing {
		return nil, fmt.Errorf("%w: only editing version can be shared (got %s)", model.ErrInvalidState, v.Status)
	}
	return svc.store.UpdateResearchVersion(id, string(model.VersionShared), nil)
}

// SupersedeVersion 基于已冻结版本创建替代版本（新版本 baseline=原版本，原版本置为 superseded）。
func (svc *Service) SupersedeVersion(id int64, code string) (*model.ResearchVersion, error) {
	if code == "" {
		return nil, fmt.Errorf("%w: code required", model.ErrInvalidInput)
	}
	old, err := svc.store.GetResearchVersion(id)
	if err != nil {
		return nil, err
	}
	if old.Status != model.VersionFrozen {
		return nil, fmt.Errorf("%w: only frozen version can be superseded (got %s)", model.ErrInvalidState, old.Status)
	}
	// 已冻结版本不可再次被替代
	if old.Status == model.VersionSuperseded {
		return nil, fmt.Errorf("%w: version already superseded", model.ErrInvalidState)
	}
	now := time.Now().UTC()
	// 先建新版本
	nv := &model.ResearchVersion{Code: code, Status: model.VersionEditing, BaselineVersionID: id}
	if err := svc.store.CreateResearchVersion(nv); err != nil {
		return nil, err
	}
	// 旧版本置为 superseded
	if _, err := svc.store.UpdateResearchVersion(id, string(model.VersionSuperseded), &now); err != nil {
		return nil, err
	}
	return nv, nil
}

// buildSnapshot 组装一个版本的不可变快照。
func (svc *Service) buildSnapshot(v *model.ResearchVersion) (version.VersionSnapshot, error) {
	snap := version.VersionSnapshot{Version: *v}
	hyps, err := svc.store.ListLineageHypotheses(v.ID)
	if err != nil {
		return snap, err
	}
	for _, h := range hyps {
		edges, err := svc.store.ListLineageEdges(h.ID)
		if err != nil {
			return snap, err
		}
		snap.Hypotheses = append(snap.Hypotheses, version.HypothesisSnapshot{Hypothesis: h, Edges: edges})
	}
	return snap, nil
}

// DiffVersions 比较当前版本与基线版本（baseID=0 表示空基线）。
func (svc *Service) DiffVersions(curID, baseID int64) (version.VersionDiff, error) {
	cur, err := svc.store.GetResearchVersion(curID)
	if err != nil {
		return version.VersionDiff{}, err
	}
	curSnap, err := svc.buildSnapshot(cur)
	if err != nil {
		return version.VersionDiff{}, err
	}
	baseSnap := version.VersionSnapshot{}
	if baseID > 0 {
		base, err := svc.store.GetResearchVersion(baseID)
		if err != nil {
			return version.VersionDiff{}, err
		}
		baseSnap, err = svc.buildSnapshot(base)
		if err != nil {
			return version.VersionDiff{}, err
		}
	}
	return version.Diff(baseSnap, curSnap), nil
}
