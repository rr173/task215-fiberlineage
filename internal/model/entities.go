// Package model 定义古籍纸张纤维谱系比对台的核心领域实体、状态枚举与值对象。
// 这些类型是无依赖的纯数据定义，供 store / 业务包 / service / httpapi 共享。
package model

import "time"

// SampleStatus 文献样本状态机：
// registered(登记) -> detecting(检测中) -> comparable(可比对) -> sealed(已封存)。
type SampleStatus string

const (
	SampleRegistered  SampleStatus = "registered"
	SampleDetecting   SampleStatus = "detecting"
	SampleComparable  SampleStatus = "comparable"
	SampleSealed      SampleStatus = "sealed"
)

// ValidSampleStatus 返回合法状态集合，用于校验。
func ValidSampleStatus(s string) bool {
	switch SampleStatus(s) {
	case SampleRegistered, SampleDetecting, SampleComparable, SampleSealed:
		return true
	default:
		return false
	}
}

// Sample 文献样本登记信息。
type Sample struct {
	ID        int64       `json:"id"`
	Code      string      `json:"code"`
	Title     string      `json:"title"`
	Source    string      `json:"source"`
	Status    SampleStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// FiberBand 纤维谱的一个波段点：band 为波段序号，intensity 为相对强度(0..1)。
type FiberBand struct {
	Band      int     `json:"band"`
	Intensity float64 `json:"intensity"`
}

// DyePeak 染料峰：wavelength 为峰位(nm)，intensity 为峰强度(0..1)。
type DyePeak struct {
	Wavelength int     `json:"wavelength"`
	Intensity  float64 `json:"intensity"`
}

// RepairLayer 修补层序：layerIndex 从外到内排序，material 为修补材料，thickness 为相对厚度(0..1)。
type RepairLayer struct {
	LayerIndex int     `json:"layer_index"`
	Material   string  `json:"material"`
	Thickness  float64 `json:"thickness"`
}

// SampleDetection 一个样本的检测谱（纤维谱 + 染料峰 + 修补层序）。
type SampleDetection struct {
	SampleID     int64        `json:"sample_id"`
	FiberSpectrum []FiberBand `json:"fiber_spectrum"`
	DyePeaks     []DyePeak    `json:"dye_peaks"`
	RepairLayers []RepairLayer `json:"repair_layers"`
	UnitKnown    bool         `json:"unit_known"` // 单位是否已知（未知则拒绝比对）
	UpdatedAt    time.Time    `json:"updated_at"`
}

// EvidenceKind 证据模态：fiber(纤维谱) / dye(染料峰) / repair(修补层序) / overall(综合)。
type EvidenceKind string

const (
	EvidenceFiber  EvidenceKind = "fiber"
	EvidenceDye    EvidenceKind = "dye"
	EvidenceRepair EvidenceKind = "repair"
	EvidenceOverall EvidenceKind = "overall"
)

// ValidEvidenceKind 校验证据模态。
func ValidEvidenceKind(s string) bool {
	switch EvidenceKind(s) {
	case EvidenceFiber, EvidenceDye, EvidenceRepair, EvidenceOverall:
		return true
	default:
		return false
	}
}

// EvidenceStatus 证据项状态机：pending(待校验) -> valid(有效) / conflict(冲突) / excluded(排除)。
type EvidenceStatus string

const (
	EvidencePending  EvidenceStatus = "pending"
	EvidenceValid    EvidenceStatus = "valid"
	EvidenceConflict EvidenceStatus = "conflict"
	EvidenceExcluded EvidenceStatus = "excluded"
)

// ValidEvidenceStatus 校验证据状态。
func ValidEvidenceStatus(s string) bool {
	switch EvidenceStatus(s) {
	case EvidencePending, EvidenceValid, EvidenceConflict, EvidenceExcluded:
		return true
	default:
		return false
	}
}

// Evidence 两个样本之间的证据项（带可信度权重）。
type Evidence struct {
	ID        int64        `json:"id"`
	SampleA   int64        `json:"sample_a"`
	SampleB   int64        `json:"sample_b"`
	Kind      EvidenceKind `json:"kind"`
	Weight    float64      `json:"weight"` // 可信度权重 0..1
	Status    EvidenceStatus `json:"status"`
	Note      string       `json:"note"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// SimilarityEdge 两样本之间的相似度边（由比对引擎按证据权重生成）。
type SimilarityEdge struct {
	ID        int64     `json:"id"`
	SampleA   int64     `json:"sample_a"`
	SampleB   int64     `json:"sample_b"`
	Score     float64   `json:"score"` // 0..1，越大越相似
	Method    string    `json:"method"`
	CreatedAt time.Time `json:"created_at"`
}

// LineageStatus 谱系假设状态机：
// draft(草稿) -> pending_evidence(待证据) -> confirmed(确认) / rejected(否决)；
// 任意阶段可进入 mutual_exclusive(互斥)。
type LineageStatus string

const (
	LineageDraft          LineageStatus = "draft"
	LineagePendingEvidence LineageStatus = "pending_evidence"
	LineageMutualExclusive LineageStatus = "mutual_exclusive"
	LineageConfirmed      LineageStatus = "confirmed"
	LineageRejected       LineageStatus = "rejected"
)

// ValidLineageStatus 校验谱系状态。
func ValidLineageStatus(s string) bool {
	switch LineageStatus(s) {
	case LineageDraft, LineagePendingEvidence, LineageMutualExclusive, LineageConfirmed, LineageRejected:
		return true
	default:
		return false
	}
}

// LineageHypothesis 谱系假设（一张假设图）。
type LineageHypothesis struct {
	ID        int64        `json:"id"`
	Code      string       `json:"code"`
	Status    LineageStatus `json:"status"`
	Note      string       `json:"note"`
	VersionID int64        `json:"version_id"` // 所属研究版本
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// LineageRelation 谱系边关系：ancestor(祖先) / descendant(后代) / same_source(同源)。
type LineageRelation string

const (
	RelationAncestor  LineageRelation = "ancestor"
	RelationDescendant LineageRelation = "descendant"
	RelationSameSource LineageRelation = "same_source"
)

// ValidLineageRelation 校验谱系边关系。
func ValidLineageRelation(s string) bool {
	switch LineageRelation(s) {
	case RelationAncestor, RelationDescendant, RelationSameSource:
		return true
	default:
		return false
	}
}

// LineageEdge 谱系假设图内的一条边。
type LineageEdge struct {
	ID           int64         `json:"id"`
	HypothesisID int64         `json:"hypothesis_id"`
	SampleA      int64         `json:"sample_a"`
	SampleB      int64         `json:"sample_b"`
	Relation     LineageRelation `json:"relation"`
	CreatedAt    time.Time     `json:"created_at"`
}

// Counterexample 谱系假设的反例（研究者记录来源谱系不成立的依据）。
type Counterexample struct {
	ID           int64     `json:"id"`
	HypothesisID int64     `json:"hypothesis_id"`
	SampleA      int64     `json:"sample_a"`
	SampleB      int64     `json:"sample_b"`
	Note         string    `json:"note"`
	CreatedAt    time.Time `json:"created_at"`
}

// VersionStatus 研究版本状态机：
// editing(编辑中) -> shared(共享) / frozen(冻结)；
// frozen 版本可被 superseded(替代)。
type VersionStatus string

const (
	VersionEditing    VersionStatus = "editing"
	VersionShared     VersionStatus = "shared"
	VersionFrozen     VersionStatus = "frozen"
	VersionSuperseded VersionStatus = "superseded"
)

// ValidVersionStatus 校验版本状态。
func ValidVersionStatus(s string) bool {
	switch VersionStatus(s) {
	case VersionEditing, VersionShared, VersionFrozen, VersionSuperseded:
		return true
	default:
		return false
	}
}

// ResearchVersion 研究版本（冻结后保留全部证据的不可变快照）。
type ResearchVersion struct {
	ID              int64        `json:"id"`
	Code            string       `json:"code"`
	Status          VersionStatus `json:"status"`
	BaselineVersionID int64      `json:"baseline_version_id"` // 差异基准，0 表示无
	FrozenAt        *time.Time   `json:"frozen_at"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}
