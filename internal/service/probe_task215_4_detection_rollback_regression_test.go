package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug04_InvalidReplacementKeepsPreviousDetection(t *testing.T) {
	svc := New(tempStore(t))
	sm, err := svc.RegisterSample("S1", "甲", "馆藏A")
	if err != nil {
		t.Fatal(err)
	}
	old := mustDet([]model.FiberBand{{Band: 1, Intensity: 0.8}}, nil, nil)
	if err := svc.SubmitDetection(sm.ID, old); err != nil {
		t.Fatal(err)
	}
	bad := &model.SampleDetection{FiberSpectrum: []model.FiberBand{{Band: 1, Intensity: 0.2}}, RepairLayers: []model.RepairLayer{{LayerIndex: 0, Material: "", Thickness: 0.3}}, UnitKnown: true}
	if err := svc.SubmitDetection(sm.ID, bad); err == nil {
		t.Fatal("invalid replacement should fail")
	}
	got, err := svc.GetDetection(sm.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.FiberSpectrum[0].Intensity != 1 {
		t.Fatalf("previous normalized detection was replaced: %+v", got.FiberSpectrum)
	}
}
