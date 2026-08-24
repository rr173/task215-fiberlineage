package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug02_SupersedeKeepsBaseline(t *testing.T) {
	svc := New(tempStore(t))
	base, err := svc.CreateVersion("V1", 0)
	if err != nil {
		t.Fatal(err)
	}
	h, err := svc.CreateLineage("H1", base.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddLineageEdge(h.ID, 1, 2, model.RelationSameSource); err != nil {
		t.Fatal(err)
	}
	if err := svc.ConfirmLineage(h.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FreezeVersion(base.ID); err != nil {
		t.Fatal(err)
	}
	next, err := svc.SupersedeVersion(base.ID, "V2")
	if err != nil {
		t.Fatal(err)
	}
	if next.BaselineVersionID != base.ID {
		t.Fatalf("replacement version baseline = %d, want %d", next.BaselineVersionID, base.ID)
	}
	old, err := svc.GetVersion(base.ID)
	if err != nil {
		t.Fatal(err)
	}
	if old.Status != model.VersionSuperseded {
		t.Fatalf("old version status = %s, want superseded", old.Status)
	}
}
