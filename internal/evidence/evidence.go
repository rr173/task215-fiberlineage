// Package evidence 封装证据项的状态机流转规则与可信度聚合辅助。
// 纯函数，无持久化依赖；service 在更新证据状态前调用 CanTransition 校验。
package evidence

import "task215-fiberlineage/internal/model"

// allowed 定义证据状态可流转关系（无向可回退，确保研究者可纠偏）。
var allowed = map[model.EvidenceStatus]map[model.EvidenceStatus]bool{
	model.EvidencePending: {
		model.EvidenceValid:    true,
		model.EvidenceConflict: true,
		model.EvidenceExcluded: true,
	},
	model.EvidenceValid: {
		model.EvidenceConflict: true,
		model.EvidenceExcluded: true,
		model.EvidencePending:  true,
	},
	model.EvidenceConflict: {
		model.EvidenceValid:    true,
		model.EvidenceExcluded: true,
		model.EvidencePending:  true,
	},
	model.EvidenceExcluded: {
		model.EvidenceValid:    true,
		model.EvidencePending:  true,
	},
}

// CanTransition 校验证据状态是否可从 from 流转到 to。
func CanTransition(from, to model.EvidenceStatus) bool {
	if from == to {
		return true
	}
	if !model.ValidEvidenceStatus(string(from)) || !model.ValidEvidenceStatus(string(to)) {
		return false
	}
	return allowed[from][to]
}

// EffectiveWeight 返回某证据在相似度合成中实际贡献的权重：
// 仅 valid 状态计入，conflict/excluded/pending 不计（pending 留给研究者校验）。
func EffectiveWeight(e model.Evidence) float64 {
	if e.Status == model.EvidenceValid {
		return e.Weight
	}
	return 0
}
