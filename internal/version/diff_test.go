package version

import (
	"task215-fiberlineage/internal/model"
	"testing"
)

func TestDiffReportsAddedHypothesisAndEdge(t *testing.T) {
	base := VersionSnapshot{}
	cur := VersionSnapshot{Hypotheses: []HypothesisSnapshot{{Hypothesis: model.LineageHypothesis{Code: "H1"}, Edges: []model.LineageEdge{{SampleA: 1, SampleB: 2, Relation: model.RelationSameSource}}}}}
	d := Diff(base, cur)
	if len(d.AddedHypotheses) != 1 || d.AddedHypotheses[0] != "H1" {
		t.Fatalf("added hypotheses = %+v", d.AddedHypotheses)
	}
	if len(d.AddedEdges) != 1 || d.AddedEdges[0] != "H1#1-2:same_source" {
		t.Fatalf("added edges = %+v", d.AddedEdges)
	}
}
