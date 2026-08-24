package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug06_FrozenVersionRejectsCounterexample(t *testing.T) {
	svc := New(tempStore(t))
	v, err := svc.CreateVersion("V1", 0)
	if err != nil {
		t.Fatal(err)
	}
	h, err := svc.CreateLineage("H1", v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddLineageEdge(h.ID, 1, 2, model.RelationSameSource); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmLineage(h.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FreezeVersion(v.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddCounterexample(h.ID, 3, 4, "contradiction"); err == nil {
		t.Fatal("frozen version must reject counterexample writes")
	}
}
