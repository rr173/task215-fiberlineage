package service

import (
	"task215-fiberlineage/internal/lineage"
	"task215-fiberlineage/internal/model"
)

// SelfCheckIssue 一条自检发现。
type SelfCheckIssue struct {
	Level   string `json:"level"`   // "error" | "warn" | "info"
	Subject string `json:"subject"` // 实体类型
	ID      int64  `json:"id"`      // 实体 ID
	Message string `json:"message"`
}

// SelfCheckReport 自检报告。
type SelfCheckReport struct {
	OK     bool             `json:"ok"`
	Issues []SelfCheckIssue `json:"issues"`
}

// SelfCheck 一致性自检：校验已确认假设不成环、互斥假设不共存确认、相似度边均指向可比对样本。
func (svc *Service) SelfCheck() SelfCheckReport {
	rep := SelfCheckReport{Issues: []SelfCheckIssue{}}
	add := func(lvl, subj string, id int64, msg string) {
		rep.Issues = append(rep.Issues, SelfCheckIssue{Level: lvl, Subject: subj, ID: id, Message: msg})
	}

	hyps, err := svc.store.ListLineageHypotheses(0)
	if err == nil {
		// 同版本互斥 + 确认冲突
		byVersion := map[int64][]model.LineageHypothesis{}
		for _, h := range hyps {
			byVersion[h.VersionID] = append(byVersion[h.VersionID], h)
		}
		for vid, list := range byVersion {
			if vid == 0 {
				continue
			}
			hasConfirmed := false
			hasMutual := false
			for _, h := range list {
				if h.Status == model.LineageConfirmed {
					hasConfirmed = true
				}
				if h.Status == model.LineageMutualExclusive {
					hasMutual = true
				}
			}
			if hasConfirmed && hasMutual {
				add("error", "version", vid, "a confirmed hypothesis coexists with a mutual-exclusive hypothesis in the same version")
			}
			// 已确认假设成环
			for _, h := range list {
				if h.Status != model.LineageConfirmed {
					continue
				}
				edges, eerr := svc.store.ListLineageEdges(h.ID)
				if eerr != nil {
					continue
				}
				if lineage.HasCycle(edges) {
					add("error", "lineage", h.ID, "confirmed hypothesis contains a cycle")
				}
			}
		}
	}

	// 相似度边指向可比对样本
	edges, err := svc.store.ListSimilarityEdges("")
	if err == nil {
		for _, e := range edges {
			for _, sid := range []int64{e.SampleA, e.SampleB} {
				s, serr := svc.store.GetSample(sid)
				if serr != nil {
					add("error", "similarity", e.ID, "edge references missing sample")
					continue
				}
				if s.Status != model.SampleComparable {
					add("warn", "similarity", e.ID, "edge references non-comparable sample")
				}
			}
		}
	}

	rep.OK = true
	for _, it := range rep.Issues {
		if it.Level == "error" {
			rep.OK = false
		}
	}
	return rep
}
