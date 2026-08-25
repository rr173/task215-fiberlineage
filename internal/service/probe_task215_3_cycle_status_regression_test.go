package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug03_CycleRejectionPreservesPendingStatus(t *testing.T) {
	svc := New(tempStore(t))
	h, err := svc.CreateLineage("H1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddLineageEdge(h.ID, 1, 2, model.RelationAncestor); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddLineageEdge(h.ID, 2, 1, model.RelationDescendant); err == nil {
		t.Fatal("cycle edge should be rejected")
	}
	got, _, _, err := svc.GetLineage(h.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.LineagePendingEvidence {
		t.Fatalf("status after rejected cycle = %s, want pending_evidence", got.Status)
	}
}
