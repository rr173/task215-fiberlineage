package service

import (
	"sync"
	"testing"

	"task215-fiberlineage/internal/model"
)

func TestBug09_ConcurrentRegistrationKeepsUniqueCode(t *testing.T) {
	svc := New(tempStore(t))
	const workers = 20
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	duplicates := 0
	otherErrors := 0
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.RegisterSample("SAME", "同码样本", "馆藏")
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				successes++
			case err != nil && model.IsDuplicate(err):
				duplicates++
			default:
				otherErrors++
			}
		}()
	}
	close(start)
	wg.Wait()
	if successes != 1 || duplicates != workers-1 || otherErrors != 0 {
		t.Fatalf("successes=%d duplicates=%d other=%d", successes, duplicates, otherErrors)
	}
}
