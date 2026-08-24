package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug01_ValidatedEvidenceRefreshesSimilarity(t *testing.T) {
	svc := New(tempStore(t))
	a, err := svc.RegisterSample("A", "甲", "馆藏A")
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.RegisterSample("B", "乙", "馆藏B")
	if err != nil {
		t.Fatal(err)
	}
	det := mustDet([]model.FiberBand{{Band: 1, Intensity: 1}}, nil, nil)
	if err := svc.SubmitDetection(a.ID, det); err != nil {
		t.Fatal(err)
	}
	if err := svc.SubmitDetection(b.ID, det); err != nil {
		t.Fatal(err)
	}
	ev, err := svc.AddEvidence(a.ID, b.ID, model.EvidenceFiber, 1, "same fiber")
	if err != nil {
		t.Fatal(err)
	}
	before, err := svc.ComputeSimilarity(a.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before.Score != 0 {
		t.Fatalf("pending evidence should contribute no confidence, got %v", before.Score)
	}
	if _, err := svc.ValidateEvidence(ev.ID, model.EvidenceValid); err != nil {
		t.Fatal(err)
	}
	edges, err := svc.ListSimilarity("weighted")
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range edges {
		if edge.SampleA == a.ID && edge.SampleB == b.ID {
			if edge.Score < 0.99 {
				t.Fatalf("validated evidence must immediately refresh similarity, got %v", edge.Score)
			}
			return
		}
	}
	t.Fatal("similarity edge disappeared after evidence validation")
}
