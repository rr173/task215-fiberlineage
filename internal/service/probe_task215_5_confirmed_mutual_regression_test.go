package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug05_ConfirmedHypothesisCannotBecomeMutualExclusive(t *testing.T) {
	svc := New(tempStore(t))
	h, err := svc.CreateLineage("H1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddLineageEdge(h.ID, 1, 2, model.RelationSameSource); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmLineage(h.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetMutualExclusive(h.ID); err == nil {
		t.Fatal("confirmed hypothesis must remain terminal")
	}
	got, _, _, err := svc.GetLineage(h.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.LineageConfirmed {
		t.Fatalf("status changed to %s", got.Status)
	}
}
