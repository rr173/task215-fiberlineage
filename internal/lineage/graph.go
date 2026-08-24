// Package lineage 提供谱系假设图（一组样本节点 + 有向关系边）的纯图算法：
// 环检测（祖先/后代不可成环）、合并相容性校验。无持久化依赖。
package lineage

import (
	"fmt"

	"task215-fiberlineage/internal/model"
)

// directedEdge 将一条谱系边转成有向 (from->to)，依据 relation：
// ancestor 表示 a 是 b 的祖先；descendant 表示 a 是 b 的后代；same_source 无方向（双向等价）。
func directedEdge(e model.LineageEdge) (int64, int64, bool) {
	switch e.Relation {
	case model.RelationAncestor:
		return e.SampleA, e.SampleB, true
	case model.RelationDescendant:
		return e.SampleB, e.SampleA, true
	case model.RelationSameSource:
		return 0, 0, false
	default:
		return 0, 0, false
	}
}

// HasCycle 检测有向谱系图是否存在环（祖先关系成环即非法）。
func HasCycle(edges []model.LineageEdge) bool {
	adj := map[int64][]int64{}
	for _, e := range edges {
		from, to, ok := directedEdge(e)
		if !ok {
			continue
		}
		adj[from] = append(adj[from], to)
	}
	const white, gray, black = 0, 1, 2
	color := map[int64]int{}
	var visit func(n int64) bool
	visit = func(n int64) bool {
		color[n] = gray
		for _, m := range adj[n] {
			switch color[m] {
			case gray:
				return true
			case white:
				if visit(m) {
					return true
				}
			}
		}
		color[n] = black
		return false
	}
	for n := range adj {
		if color[n] == white {
			if visit(n) {
				return true
			}
		}
	}
	return false
}

// ConflictingRelations 检查两条边对同一对样本是否给出互相矛盾的关系。
// 例如 a->ancestor(b) 与 a->descendant(b) 矛盾；same_source 与有向关系不矛盾。
func ConflictingRelations(a, b model.LineageEdge) bool {
	if a.SampleA != b.SampleA || a.SampleB != b.SampleB {
		if a.SampleA != b.SampleB || a.SampleB != b.SampleA {
			return false
		}
	}
	ra, rb := a.Relation, b.Relation
	if ra == rb {
		return false
	}
	// ancestor 与 descendant 反向视为矛盾
	if (ra == model.RelationAncestor && rb == model.RelationDescendant) ||
		(ra == model.RelationDescendant && rb == model.RelationAncestor) {
		return true
	}
	return false
}

// CanMerge 校验两个假设的边集能否合并：节点集合不得相交，否则会产生跨假设歧义。
func CanMerge(h1Edges, h2Edges []model.LineageEdge) error {
	nset := map[int64]bool{}
	for _, e := range h1Edges {
		nset[e.SampleA] = true
		nset[e.SampleB] = true
	}
	for _, e := range h2Edges {
		if nset[e.SampleA] || nset[e.SampleB] {
			return fmt.Errorf("%w: hypotheses share samples, cannot merge", model.ErrConflict)
		}
	}
	return nil
}
