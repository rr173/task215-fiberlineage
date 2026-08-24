package service

import (
	"fmt"

	"task215-fiberlineage/internal/compare"
	"task215-fiberlineage/internal/model"
)

// recomputePair 重新计算并写入两个样本之间的相似度边（按证据权重合成）。
func (svc *Service) recomputePair(sampleA, sampleB int64) error {
	if sampleA == sampleB {
		return fmt.Errorf("%w: cannot compare sample with itself", model.ErrInvalidInput)
	}
	da, errA := svc.store.GetDetection(sampleA)
	db, errB := svc.store.GetDetection(sampleB)
	if errA != nil || errB != nil {
		// 任一样本无检测谱：移除既有相似度边
		_ = svc.store.DeleteSimilarityEdges()
		// 重新载入有效对避免误删全局；这里仅当确实都不可比时跳过
		if errA != nil && errB != nil {
			return nil
		}
		return fmt.Errorf("%w: both samples need detection for similarity", model.ErrInvalidInput)
	}
	a, b := orderPair(sampleA, sampleB)
	evs, err := svc.store.ListEvidence("", "")
	if err != nil {
		return fmt.Errorf("list evidence: %w", err)
	}
	var pair []model.Evidence
	for _, e := range evs {
		ea, eb := orderPair(e.SampleA, e.SampleB)
		if ea == a && eb == b {
			pair = append(pair, e)
		}
	}
	w, factor := compare.AggregateWeights(pair)
	score := compare.Overall(da, db, w, factor)
	edge := &model.SimilarityEdge{SampleA: a, SampleB: b, Score: score, Method: "weighted"}
	return svc.store.UpsertSimilarityEdge(edge)
}

// ComputeSimilarity 显式计算并返回两样本相似度（同时持久化边）。
func (svc *Service) ComputeSimilarity(sampleA, sampleB int64) (*model.SimilarityEdge, error) {
	if err := svc.recomputePair(sampleA, sampleB); err != nil {
		return nil, err
	}
	a, b := orderPair(sampleA, sampleB)
	edges, err := svc.store.ListSimilarityEdges("weighted")
	if err != nil {
		return nil, err
	}
	for i := range edges {
		if edges[i].SampleA == a && edges[i].SampleB == b {
			return &edges[i], nil
		}
	}
	return nil, model.ErrNotFound
}

// RecomputeAllSimilarities 对所有可比对样本对重算相似度边。
func (svc *Service) RecomputeAllSimilarities() (int, error) {
	samples, err := svc.store.ListSamples()
	if err != nil {
		return 0, err
	}
	var comparable []model.Sample
	for _, s := range samples {
		if s.Status == model.SampleComparable {
			comparable = append(comparable, s)
		}
	}
	if err := svc.store.DeleteSimilarityEdges(); err != nil {
		return 0, err
	}
	n := 0
	for i := 0; i < len(comparable); i++ {
		for j := i + 1; j < len(comparable); j++ {
			if err := svc.recomputePair(comparable[i].ID, comparable[j].ID); err != nil {
				continue
			}
			n++
		}
	}
	return n, nil
}

// ListSimilarity 列出相似度边（按 method 过滤，空串取全部）。
func (svc *Service) ListSimilarity(method string) ([]model.SimilarityEdge, error) {
	return svc.store.ListSimilarityEdges(method)
}

// orderPair 规整无向对顺序（与 store 一致）。
func orderPair(a, b int64) (int64, int64) {
	if a <= b {
		return a, b
	}
	return b, a
}
