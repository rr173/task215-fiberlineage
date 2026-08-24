package lineage

import (
	"task215-fiberlineage/internal/model"
	"testing"
)

func TestHasCycle(t *testing.T) {
	acyclic := []model.LineageEdge{{SampleA: 1, SampleB: 2, Relation: model.RelationAncestor}, {SampleA: 2, SampleB: 3, Relation: model.RelationAncestor}}
	if HasCycle(acyclic) {
		t.Fatal("acyclic graph reported as cyclic")
	}
	cyclic := append(acyclic, model.LineageEdge{SampleA: 3, SampleB: 1, Relation: model.RelationAncestor})
	if !HasCycle(cyclic) {
		t.Fatal("cycle was not detected")
	}
}

func TestCanMergeRejectsSharedSample(t *testing.T) {
	err := CanMerge([]model.LineageEdge{{SampleA: 1, SampleB: 2}}, []model.LineageEdge{{SampleA: 2, SampleB: 3}})
	if err == nil || !model.IsConflict(err) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
