package service

import (
	"testing"
	"time"
)

func TestBug10_SelfCheckReportsDanglingSimilarity(t *testing.T) {
	svc := New(tempStore(t))
	stamp := time.Now().UTC().Format(time.RFC3339)
	if _, err := svc.Store().DB().Exec(`INSERT INTO similarity_edges(sample_a,sample_b,score,method,created_at) VALUES(?,?,?,?,?)`, 9001, 9002, 0.5, "weighted", stamp); err != nil {
		t.Fatal(err)
	}
	rep := svc.SelfCheck()
	if rep.OK {
		t.Fatal("self-check must reject similarity edges that reference missing samples")
	}
}
