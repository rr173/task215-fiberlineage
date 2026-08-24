// Package version 提供研究版本的快照构建与差异比对（冻结/替代版本的不可变视图对比）。
// 纯函数，输入由 service 从 store 组装。
package version

import "task215-fiberlineage/internal/model"

// HypothesisSnapshot 一个谱系假设及其边集合的快照。
type HypothesisSnapshot struct {
	Hypothesis model.LineageHypothesis
	Edges      []model.LineageEdge
}

// VersionSnapshot 一个研究版本的不可变快照（含其下全部谱系假设与边）。
type VersionSnapshot struct {
	Version     model.ResearchVersion
	Hypotheses  []HypothesisSnapshot
}

// VersionDiff 两版本之间的差异。
type VersionDiff struct {
	AddedHypotheses   []string `json:"added_hypotheses"`
	RemovedHypotheses []string `json:"removed_hypotheses"`
	AddedEdges        []string `json:"added_edges"`
	RemovedEdges      []string `json:"removed_edges"`
	ChangedHypotheses []string `json:"changed_hypotheses"`
}

// edgeKey 生成边的稳定标识（假设 code + 样本对 + 关系）。
func edgeKey(hypCode string, e model.LineageEdge) string {
	return hypCode + "#" + itoa(e.SampleA) + "-" + itoa(e.SampleB) + ":" + string(e.Relation)
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Diff 比较基线版本与当前版本，返回新增/移除/变更的假设与边。
func Diff(base, cur VersionSnapshot) VersionDiff {
	d := VersionDiff{}
	baseH := map[string]HypothesisSnapshot{}
	for _, h := range base.Hypotheses {
		baseH[h.Hypothesis.Code] = h
	}
	curH := map[string]HypothesisSnapshot{}
	for _, h := range cur.Hypotheses {
		curH[h.Hypothesis.Code] = h
	}
	for code, h := range curH {
		if _, ok := baseH[code]; !ok {
			d.AddedHypotheses = append(d.AddedHypotheses, code)
			for _, e := range h.Edges {
				d.AddedEdges = append(d.AddedEdges, edgeKey(code, e))
			}
			continue
		}
		// 同名假设：比对状态与边
		bh := baseH[code]
		if bh.Hypothesis.Status != h.Hypothesis.Status || bh.Hypothesis.Note != h.Hypothesis.Note {
			d.ChangedHypotheses = append(d.ChangedHypotheses, code)
		}
		baseEdges := map[string]bool{}
		for _, e := range bh.Edges {
			baseEdges[edgeKey(code, e)] = true
		}
		for _, e := range h.Edges {
			k := edgeKey(code, e)
			if !baseEdges[k] {
				d.AddedEdges = append(d.AddedEdges, k)
			}
		}
	}
	for code, h := range baseH {
		if _, ok := curH[code]; !ok {
			d.RemovedHypotheses = append(d.RemovedHypotheses, code)
			for _, e := range h.Edges {
				d.RemovedEdges = append(d.RemovedEdges, edgeKey(code, e))
			}
		}
	}
	return d
}
