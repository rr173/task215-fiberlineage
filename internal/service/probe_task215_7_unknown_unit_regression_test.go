package service

import (
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug07_UnknownUnitsRejectDetectionBeforeWrite(t *testing.T) {
	svc := New(tempStore(t))
	sm, err := svc.RegisterSample("S1", "甲", "馆藏A")
	if err != nil {
		t.Fatal(err)
	}
	det := &model.SampleDetection{FiberSpectrum: []model.FiberBand{{Band: 1, Intensity: 0.5}}, UnitKnown: false}
	if err := svc.SubmitDetection(sm.ID, det); err == nil {
		t.Fatal("unknown detection units must be rejected")
	}
	if _, err := svc.GetDetection(sm.ID); err == nil {
		t.Fatal("rejected detection must not be persisted")
	}
}
