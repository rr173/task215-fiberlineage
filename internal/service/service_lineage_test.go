package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

// TestSetMutualExclusiveRejectedOnConfirmedHypothesis 锁定终态约束：
// 一个谱系假设被确认后即为终局结论，不得再被标记为互斥；
// 非法状态转换应被拒绝，且原 confirmed 状态被保留。
func TestSetMutualExclusiveRejectedOnConfirmedHypothesis(t *testing.T) {
	svc := New(tempStore(t))

	// 建版本 + 假设，加一条边后确认。
	v, err := svc.CreateVersion("V-MTX", 0)
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	h, err := svc.CreateLineage("H-MTX", v.ID)
	if err != nil {
		t.Fatalf("create lineage: %v", err)
	}
	if _, err := svc.AddLineageEdge(h.ID, 1, 2, model.RelationSameSource); err != nil {
		t.Fatalf("add edge: %v", err)
	}
	if err := svc.ConfirmLineage(h.ID); err != nil {
		t.Fatalf("confirm lineage: %v", err)
	}

	// 确认后再标互斥必须被拒，且为状态机非法（ErrInvalidState）。
	err = svc.SetMutualExclusive(h.ID)
	if err == nil {
		t.Fatal("expected SetMutualExclusive on a confirmed hypothesis to be rejected")
	}
	if !model.IsInvalidState(err) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}

	// 终态被保留：重新读取假设仍为 confirmed。
	got, err := svc.store.GetLineageHypothesis(h.ID)
	if err != nil {
		t.Fatalf("reload hypothesis: %v", err)
	}
	if got.Status != model.LineageConfirmed {
		t.Fatalf("expected confirmed status preserved, got %s", got.Status)
	}
	if got.Note == "mutual-exclusive" {
		t.Fatalf("confirmed hypothesis note must not be clobbered to mutual-exclusive, got %q", got.Note)
	}

	// 全局自检仍应全绿（确认 + 互斥不共存的不变量未被破坏）。
	if rep := svc.SelfCheck(); !rep.OK {
		t.Fatalf("self-check not ok after rejected transition: %+v", rep.Issues)
	}
}

// TestSetMutualExclusiveRejectedOnRejectedHypothesis 否决同为终态，
// 同样不可回退为互斥。
func TestSetMutualExclusiveRejectedOnRejectedHypothesis(t *testing.T) {
	svc := New(tempStore(t))

	v, err := svc.CreateVersion("V-REJ", 0)
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	h, err := svc.CreateLineage("H-REJ", v.ID)
	if err != nil {
		t.Fatalf("create lineage: %v", err)
	}
	if _, err := svc.AddLineageEdge(h.ID, 1, 2, model.RelationSameSource); err != nil {
		t.Fatalf("add edge: %v", err)
	}
	if err := svc.RejectLineage(h.ID); err != nil {
		t.Fatalf("reject lineage: %v", err)
	}

	if err := svc.SetMutualExclusive(h.ID); err == nil || !model.IsInvalidState(err) {
		t.Fatalf("expected ErrInvalidState for mutual-exclusive on rejected hypothesis, got %v", err)
	}

	got, err := svc.store.GetLineageHypothesis(h.ID)
	if err != nil {
		t.Fatalf("reload hypothesis: %v", err)
	}
	if got.Status != model.LineageRejected {
		t.Fatalf("expected rejected status preserved, got %s", got.Status)
	}
}

// TestSetMutualExclusiveAllowedOnNonTerminal 草稿/待证据阶段标记互斥应正常通过。
func TestSetMutualExclusiveAllowedOnNonTerminal(t *testing.T) {
	svc := New(tempStore(t))

	v, err := svc.CreateVersion("V-OK", 0)
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	h, err := svc.CreateLineage("H-OK", v.ID)
	if err != nil {
		t.Fatalf("create lineage: %v", err)
	}
	// 首条边后进入 pending_evidence；标记互斥应成功。
	if _, err := svc.AddLineageEdge(h.ID, 1, 2, model.RelationSameSource); err != nil {
		t.Fatalf("add edge: %v", err)
	}
	if err := svc.SetMutualExclusive(h.ID); err != nil {
		t.Fatalf("expected SetMutualExclusive on pending_evidence to succeed, got %v", err)
	}
	got, err := svc.store.GetLineageHypothesis(h.ID)
	if err != nil {
		t.Fatalf("reload hypothesis: %v", err)
	}
	if got.Status != model.LineageMutualExclusive {
		t.Fatalf("expected mutual_exclusive status, got %s", got.Status)
	}
}
