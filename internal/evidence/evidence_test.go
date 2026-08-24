package evidence

import (
	"task215-fiberlineage/internal/model"
	"testing"
)

func TestCanTransition(t *testing.T) {
	valid := [][2]model.EvidenceStatus{{model.EvidencePending, model.EvidenceValid}, {model.EvidenceValid, model.EvidenceConflict}, {model.EvidenceConflict, model.EvidenceExcluded}}
	for _, pair := range valid {
		if !CanTransition(pair[0], pair[1]) {
			t.Fatalf("%s -> %s should be allowed", pair[0], pair[1])
		}
	}
	if CanTransition(model.EvidencePending, model.EvidenceStatus("unknown")) {
		t.Fatal("unknown status should be rejected")
	}
}

func TestEffectiveWeightOnlyCountsValid(t *testing.T) {
	if got := EffectiveWeight(model.Evidence{Weight: .7, Status: model.EvidencePending}); got != 0 {
		t.Fatalf("pending weight = %v", got)
	}
	if got := EffectiveWeight(model.Evidence{Weight: .7, Status: model.EvidenceValid}); got != .7 {
		t.Fatalf("valid weight = %v", got)
	}
}
