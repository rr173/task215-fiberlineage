package compare

import (
	"task215-fiberlineage/internal/model"
	"testing"
)

func TestModalSimilarities(t *testing.T) {
	fiber := []model.FiberBand{{Band: 1, Intensity: 0.8}, {Band: 2, Intensity: 0.4}}
	if got := FiberSimilarity(fiber, fiber); got != 1 {
		t.Fatalf("fiber similarity = %v, want 1", got)
	}
	if got := DyeSimilarity([]model.DyePeak{{Wavelength: 500, Intensity: 1}}, []model.DyePeak{{Wavelength: 500, Intensity: 0}}); got != 0 {
		t.Fatalf("dye similarity = %v, want 0", got)
	}
	if got := RepairSimilarity([]model.RepairLayer{{LayerIndex: 1, Material: "hemp", Thickness: .4}}, []model.RepairLayer{{LayerIndex: 1, Material: "hemp", Thickness: .4}}); got != 1 {
		t.Fatalf("repair similarity = %v, want 1", got)
	}
}

func TestOverallUsesEvidenceFactor(t *testing.T) {
	d := &model.SampleDetection{FiberSpectrum: []model.FiberBand{{Band: 1, Intensity: 1}}}
	if got := Overall(d, d, FiberWeights{Fiber: 1}, .25); got != .25 {
		t.Fatalf("overall = %v, want .25", got)
	}
	w, factor := AggregateWeights([]model.Evidence{{Kind: model.EvidenceFiber, Weight: .8, Status: model.EvidenceValid}, {Kind: model.EvidenceDye, Weight: .2, Status: model.EvidenceConflict}})
	if w.Fiber != .8 || factor != .8 {
		t.Fatalf("weights = %+v factor=%v", w, factor)
	}
}
