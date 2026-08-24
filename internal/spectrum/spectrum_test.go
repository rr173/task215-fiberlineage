package spectrum

import (
	"task215-fiberlineage/internal/model"
	"testing"
)

func TestValidateRejectsUnknownUnitsAndDuplicates(t *testing.T) {
	if err := Validate(&model.SampleDetection{UnitKnown: false}); err == nil {
		t.Fatal("unknown units should be rejected")
	}
	if err := Validate(&model.SampleDetection{UnitKnown: true, FiberSpectrum: []model.FiberBand{{Band: 1, Intensity: .2}, {Band: 1, Intensity: .3}}}); err == nil {
		t.Fatal("duplicate bands should be rejected")
	}
}

func TestNormalizeDoesNotMutateInput(t *testing.T) {
	in := model.SampleDetection{FiberSpectrum: []model.FiberBand{{Band: 1, Intensity: .2}, {Band: 2, Intensity: .4}}}
	out := NormalizeIntensities(in)
	if in.FiberSpectrum[0].Intensity != .2 {
		t.Fatalf("input mutated: %+v", in.FiberSpectrum)
	}
	if out.FiberSpectrum[1].Intensity != 1 {
		t.Fatalf("normalized max = %v, want 1", out.FiberSpectrum[1].Intensity)
	}
}
