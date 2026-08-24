package service

import (
	"fmt"

	"task215-fiberlineage/internal/lineage"
	"task215-fiberlineage/internal/model"
)

// CreateLineage 在研究版本下创建谱系假设（草稿）。
func (svc *Service) CreateLineage(code string, versionID int64) (*model.LineageHypothesis, error) {
	if code == "" {
		return nil, fmt.Errorf("%w: code required", model.ErrInvalidInput)
	}
	if versionID > 0 {
		v, err := svc.store.GetResearchVersion(versionID)
		if err != nil {
			return nil, err
		}
		if v.Status != model.VersionEditing {
			return nil, fmt.Errorf("%w: lineage can only be created in an editing version", model.ErrForbidden)
		}
	}
	h := &model.LineageHypothesis{Code: code, Status: model.LineageDraft, VersionID: versionID}
	if err := svc.store.CreateLineageHypothesis(h); err != nil {
		return nil, err
	}
	return h, nil
}

// GetLineage 获取假设及其全部边与反例。
func (svc *Service) GetLineage(id int64) (*model.LineageHypothesis, []model.LineageEdge, []model.Counterexample, error) {
	h, err := svc.store.GetLineageHypothesis(id)
	if err != nil {
		return nil, nil, nil, err
	}
	edges, err := svc.store.ListLineageEdges(id)
	if err != nil {
		return nil, nil, nil, err
	}
	cxs, err := svc.store.ListCounterexamples(id)
	if err != nil {
		return nil, nil, nil, err
	}
	return h, edges, cxs, nil
}

// ListLineages 列出假设（按版本过滤）。
func (svc *Service) ListLineages(versionID int64) ([]model.LineageHypothesis, error) {
	return svc.store.ListLineageHypotheses(versionID)
}

// versionEditable 校验假设所属版本是否处于编辑中（仅编辑中可改图）。
func (svc *Service) versionEditable(hypID int64) error {
	h, err := svc.store.GetLineageHypothesis(hypID)
	if err != nil {
		return err
	}
	if h.VersionID > 0 {
		v, err := svc.store.GetResearchVersion(h.VersionID)
		if err != nil {
			return err
		}
		if v.Status != model.VersionEditing {
			return fmt.Errorf("%w: version %d is %s, hypothesis graph is immutable", model.ErrForbidden, v.ID, v.Status)
		}
	}
	return nil
}

// AddLineageEdge 在假设图中加边：要求版本可编辑、假设未冻结为终态，并校验不成环/不矛盾。
func (svc *Service) AddLineageEdge(hypID, a, b int64, relation model.LineageRelation) (*model.LineageEdge, error) {
	if err := svc.versionEditable(hypID); err != nil {
		return nil, err
	}
	h, err := svc.store.GetLineageHypothesis(hypID)
	if err != nil {
		return nil, err
	}
	if h.Status == model.LineageConfirmed || h.Status == model.LineageRejected {
		return nil, fmt.Errorf("%w: cannot edit a %s hypothesis", model.ErrForbidden, h.Status)
	}
	edge := &model.LineageEdge{HypothesisID: hypID, SampleA: a, SampleB: b, Relation: relation}
	if err := svc.store.AddLineageEdge(edge); err != nil {
		return nil, err
	}
	// 成环/矛盾校验
	edges, err := svc.store.ListLineageEdges(hypID)
	if err != nil {
		return nil, err
	}
	if lineage.HasCycle(edges) {
		_ = svc.store.DeleteLineageEdge(edge.ID)
		_, _ = svc.store.UpdateLineageHypothesis(hypID, string(model.LineageConfirmed), "")
		return nil, fmt.Errorf("%w: lineage edge would create a cycle", model.ErrConflict)
	}
	for _, e := range edges {
		if e.ID == edge.ID {
			continue
		}
		if lineage.ConflictingRelations(e, *edge) {
			_ = svc.store.DeleteLineageEdge(edge.ID)
			return nil, fmt.Errorf("%w: conflicting relation on same sample pair", model.ErrConflict)
		}
	}
	// 首条边后推进草稿 -> 待证据
	if h.Status == model.LineageDraft {
		if _, err := svc.store.UpdateLineageHypothesis(hypID, string(model.LineageConfirmed), ""); err != nil {
			return nil, err
		}
	}
	return edge, nil
}

// SplitLineageEdge 拆分谱系：删除一条边（版本可编辑、假设非终态）。
func (svc *Service) SplitLineageEdge(edgeID int64) error {
	// 先拿到边以校验版本可编辑
	edges, err := svc.allEdgesFor(edgeID)
	if err != nil {
		return err
	}
	_ = edges
	if err := svc.deleteEdgeWithCheck(edgeID); err != nil {
		return err
	}
	return nil
}

// MergeLineages 将 src 假设合并进 dst：要求同版本、可编辑、节点不相交。
func (svc *Service) MergeLineages(dstID, srcID int64) error {
	if dstID == srcID {
		return fmt.Errorf("%w: cannot merge hypothesis into itself", model.ErrInvalidInput)
	}
	if err := svc.versionEditable(dstID); err != nil {
		return err
	}
	dst, err := svc.store.GetLineageHypothesis(dstID)
	if err != nil {
		return err
	}
	src, err := svc.store.GetLineageHypothesis(srcID)
	if err != nil {
		return err
	}
	if dst.VersionID != src.VersionID {
		return fmt.Errorf("%w: hypotheses belong to different versions", model.ErrInvalidInput)
	}
	dstEdges, err := svc.store.ListLineageEdges(dstID)
	if err != nil {
		return err
	}
	srcEdges, err := svc.store.ListLineageEdges(srcID)
	if err != nil {
		return err
	}
	if err := lineage.CanMerge(dstEdges, srcEdges); err != nil {
		return err
	}
	for _, e := range srcEdges {
		if _, err := svc.AddLineageEdge(dstID, e.SampleA, e.SampleB, e.Relation); err != nil {
			return fmt.Errorf("merge edge: %w", err)
		}
	}
	return svc.store.DeleteHypothesisCascade(srcID)
}

// ConfirmLineage 确认假设：要求版本可编辑、非终态、至少有一个边或反例，且不与其它互斥假设同时确认。
func (svc *Service) ConfirmLineage(hypID int64) error {
	if err := svc.versionEditable(hypID); err != nil {
		return err
	}
	h, err := svc.store.GetLineageHypothesis(hypID)
	if err != nil {
		return err
	}
	if h.Status == model.LineageConfirmed {
		return nil
	}
	if h.Status == model.LineageRejected {
		return fmt.Errorf("%w: cannot confirm a rejected hypothesis", model.ErrInvalidState)
	}
	edges, err := svc.store.ListLineageEdges(hypID)
	if err != nil {
		return err
	}
	cxs, err := svc.store.ListCounterexamples(hypID)
	if err != nil {
		return err
	}
	if len(edges) == 0 && len(cxs) == 0 {
		return fmt.Errorf("%w: hypothesis needs at least one edge or counterexample to confirm", model.ErrInvalidInput)
	}
	if lineage.HasCycle(edges) {
		return fmt.Errorf("%w: hypothesis graph has a cycle, cannot confirm", model.ErrConflict)
	}
	// 互斥约束：同版本内若存在 mutual_exclusive 假设（作为另一种结论候选），不得再确认本假设。
	sibs, err := svc.store.ListLineageHypotheses(h.VersionID)
	if err != nil {
		return err
	}
	for _, s := range sibs {
		if s.ID == hypID {
			continue
		}
		if s.Status == model.LineageMutualExclusive {
			return fmt.Errorf("%w: a mutual-exclusive hypothesis exists in the same version", model.ErrConflict)
		}
	}
	_, err = svc.store.UpdateLineageHypothesis(hypID, string(model.LineageConfirmed), "")
	return err
}

// RejectLineage 否决假设。
func (svc *Service) RejectLineage(hypID int64) error {
	if err := svc.versionEditable(hypID); err != nil {
		return err
	}
	h, err := svc.store.GetLineageHypothesis(hypID)
	if err != nil {
		return err
	}
	if h.Status == model.LineageConfirmed {
		return fmt.Errorf("%w: cannot reject a confirmed hypothesis", model.ErrInvalidState)
	}
	_, err = svc.store.UpdateLineageHypothesis(hypID, string(model.LineageRejected), "")
	return err
}

// SetMutualExclusive 将假设标记为互斥（仅草稿/待证据可标记）。
func (svc *Service) SetMutualExclusive(hypID int64) error {
	if err := svc.versionEditable(hypID); err != nil {
		return err
	}
	h, err := svc.store.GetLineageHypothesis(hypID)
	if err != nil {
		return err
	}
	if h.Status == model.LineageConfirmed || h.Status == model.LineageRejected {
		return fmt.Errorf("%w: cannot mark terminal hypothesis as mutual-exclusive", model.ErrInvalidState)
	}
	_, err = svc.store.UpdateLineageHypothesis(hypID, string(model.LineageMutualExclusive), "")
	return err
}

// AddCounterexample 给假设加反例。
func (svc *Service) AddCounterexample(hypID, a, b int64, note string) (*model.Counterexample, error) {
	if err := svc.versionEditable(hypID); err != nil {
		return nil, err
	}
	c := &model.Counterexample{HypothesisID: hypID, SampleA: a, SampleB: b, Note: note}
	if err := svc.store.AddCounterexample(c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListCounterexamples 列出反例。
func (svc *Service) ListCounterexamples(hypID int64) ([]model.Counterexample, error) {
	return svc.store.ListCounterexamples(hypID)
}

// allEdgesFor / deleteEdgeWithCheck 是 SplitLineageEdge 的辅助：通过边定位假设并校验可编辑。
func (svc *Service) allEdgesFor(edgeID int64) ([]model.LineageEdge, error) {
	// 通过遍历全部假设边定位该边所属假设
	hyps, err := svc.store.ListLineageHypotheses(0)
	if err != nil {
		return nil, err
	}
	for _, h := range hyps {
		edges, err := svc.store.ListLineageEdges(h.ID)
		if err != nil {
			return nil, err
		}
		for _, e := range edges {
			if e.ID == edgeID {
				if err := svc.versionEditable(h.ID); err != nil {
					return nil, err
				}
				hh, err := svc.store.GetLineageHypothesis(h.ID)
				if err != nil {
					return nil, err
				}
				if hh.Status == model.LineageConfirmed || hh.Status == model.LineageRejected {
					return nil, fmt.Errorf("%w: cannot split edge of a %s hypothesis", model.ErrForbidden, hh.Status)
				}
				return edges, nil
			}
		}
	}
	return nil, model.ErrNotFound
}

func (svc *Service) deleteEdgeWithCheck(edgeID int64) error {
	if err := svc.store.DeleteLineageEdge(edgeID); err != nil {
		return err
	}
	return nil
}
