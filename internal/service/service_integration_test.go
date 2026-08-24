package service

import (
	"path/filepath"
	"testing"

	"task215-fiberlineage/internal/model"
	"task215-fiberlineage/internal/store"
)

// tempStore 在临时目录建库并迁移，返回 store 与清理函数。
func tempStore(t *testing.T) *store.Store {
	t.Helper()
	db := filepath.Join(t.TempDir(), "it.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func mustDet(fiber, dye []model.FiberBand, dyePeaks []model.DyePeak) *model.SampleDetection {
	return &model.SampleDetection{
		FiberSpectrum: fiber,
		DyePeaks:      dyePeaks,
		RepairLayers:  nil,
		UnitKnown:     true,
	}
}

func TestFullLoop(t *testing.T) {
	svc := New(tempStore(t))

	// 1) 登记两个样本
	s1, err := svc.RegisterSample("S1", "宋刻本甲", "馆藏A")
	if err != nil {
		t.Fatalf("register s1: %v", err)
	}
	s2, err := svc.RegisterSample("S2", "宋刻本乙", "馆藏B")
	if err != nil {
		t.Fatalf("register s2: %v", err)
	}

	// 2) 提交检测谱（相同纤维谱 -> 高相似）
	fb := []model.FiberBand{{Band: 1, Intensity: 0.9}, {Band: 2, Intensity: 0.4}}
	if err := svc.SubmitDetection(s1.ID, mustDet(fb, nil, nil)); err != nil {
		t.Fatalf("detect s1: %v", err)
	}
	if err := svc.SubmitDetection(s2.ID, mustDet(fb, nil, nil)); err != nil {
		t.Fatalf("detect s2: %v", err)
	}

	// 3) 加证据并校验
	ev, err := svc.AddEvidence(s1.ID, s2.ID, model.EvidenceFiber, 0.8, "纤维一致")
	if err != nil {
		t.Fatalf("add evidence: %v", err)
	}
	if _, err := svc.ValidateEvidence(ev.ID, model.EvidenceValid); err != nil {
		t.Fatalf("validate evidence: %v", err)
	}

	// 4) 计算相似度（应 > 0）
	edge, err := svc.ComputeSimilarity(s1.ID, s2.ID)
	if err != nil {
		t.Fatalf("compute similarity: %v", err)
	}
	if edge.Score <= 0 {
		t.Fatalf("expected positive similarity, got %v", edge.Score)
	}

	// 5) 研究版本 + 谱系假设
	v, err := svc.CreateVersion("V1", 0)
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	h, err := svc.CreateLineage("H1", v.ID)
	if err != nil {
		t.Fatalf("create lineage: %v", err)
	}
	if _, err := svc.AddLineageEdge(h.ID, s1.ID, s2.ID, model.RelationSameSource); err != nil {
		t.Fatalf("add edge: %v", err)
	}
	if err := svc.ConfirmLineage(h.ID); err != nil {
		t.Fatalf("confirm lineage: %v", err)
	}
	if _, err := svc.FreezeVersion(v.ID); err != nil {
		t.Fatalf("freeze version: %v", err)
	}

	// 6) 冻结后写操作应被拒绝
	if _, err := svc.AddLineageEdge(h.ID, s1.ID, s1.ID, model.RelationAncestor); err == nil {
		t.Fatalf("expected forbidden edge on frozen version")
	}

	// 7) 自检通过
	rep := svc.SelfCheck()
	if !rep.OK {
		t.Fatalf("self-check not ok: %+v", rep.Issues)
	}
}
