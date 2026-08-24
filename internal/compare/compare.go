// Package compare 实现古籍样本之间的相似度计算引擎：
// 分别针对纤维谱、染料峰、修补层序计算相似度，再按证据权重合成综合相似度。
// 纯函数，输入为已规整的检测谱，不依赖数据库。
package compare

import (
	"math"

	"task215-fiberlineage/internal/model"
)

// FiberSimilarity 纤维谱余弦相似度：按波段对齐强度向量。
func FiberSimilarity(a, b []model.FiberBand) float64 {
	va, vb := toVec(bandsToMap(a), bandsToMap(b))
	return cosine(va, vb)
}

// DyeSimilarity 染料峰余弦相似度：按峰位对齐强度向量。
func DyeSimilarity(a, b []model.DyePeak) float64 {
	va, vb := toVec(peaksToMap(a), peaksToMap(b))
	return cosine(va, vb)
}

// RepairSimilarity 修补层序相似度：按层序号比对材料是否一致及厚度接近度。
func RepairSimilarity(a, b []model.RepairLayer) float64 {
	ma, mb := layersToMap(a), layersToMap(b)
	keys := unionKeys(ma, mb)
	if len(keys) == 0 {
		return 0
	}
	sum := 0.0
	for _, k := range keys {
		la, oka := ma[k]
		lb, okb := mb[k]
		if !oka || !okb {
			continue // 单边缺失，不贡献相似
		}
		match := 0.0
		if la.Material == lb.Material {
			match += 0.6
		}
		match += 0.4 * (1 - math.Abs(la.Thickness-lb.Thickness))
		if match > 1 {
			match = 1
		}
		sum += match
	}
	if sum == 0 {
		return 0
	}
	return clamp01(sum / float64(len(keys)))
}

// Overall 综合相似度：fiber/dye/repair 三模态按证据权重合成。
// weights 为各模态权重（0..1），缺失模态权重取 0；结果再乘以 evidenceFactor（0..1，证据可信度）。
func Overall(a, b *model.SampleDetection, w FiberWeights, evidenceFactor float64) float64 {
	fs := FiberSimilarity(a.FiberSpectrum, b.FiberSpectrum)
	ds := 0.0
	if len(a.DyePeaks) > 0 && len(b.DyePeaks) > 0 {
		ds = DyeSimilarity(a.DyePeaks, b.DyePeaks)
	}
	rs := 0.0
	if len(a.RepairLayers) > 0 && len(b.RepairLayers) > 0 {
		rs = RepairSimilarity(a.RepairLayers, b.RepairLayers)
	}
	total := w.Fiber + w.Dye + w.Repair
	if total <= 0 {
		// 无证据权重时等权
		w.Fiber, w.Dye, w.Repair = 1, 1, 1
		total = 3
	}
	combined := (fs*w.Fiber + ds*w.Dye + rs*w.Repair) / total
	return clamp01(combined * evidenceFactor)
}

// FiberWeights 各模态的合成权重。
type FiberWeights struct {
	Fiber float64
	Dye   float64
	Repair float64
}

// AggregateWeights 将一对样本的证据项聚合为 (各模态权重, 证据系数)。
// 证据系数 = Σ weight(有效) / max(1, Σ weight(全部有效/冲突))，无有效证据时取 0.5 作为中性基线。
func AggregateWeights(evidences []model.Evidence) (FiberWeights, float64) {
	w := FiberWeights{}
	var validSum, allSum float64
	for _, e := range evidences {
		if e.Status == model.EvidenceExcluded {
			continue
		}
		allSum += e.Weight
		if e.Status != model.EvidenceValid {
			continue
		}
		validSum += e.Weight
		switch e.Kind {
		case model.EvidenceFiber:
			w.Fiber += e.Weight
		case model.EvidenceDye:
			w.Dye += e.Weight
		case model.EvidenceRepair:
			w.Repair += e.Weight
		case model.EvidenceOverall:
			w.Fiber += e.Weight
			w.Dye += e.Weight
			w.Repair += e.Weight
		}
	}
	factor := 0.5
	if allSum > 0 {
		factor = clamp01(validSum / allSum)
	}
	return w, factor
}

// ---- 向量辅助 ----

func bandsToMap(in []model.FiberBand) map[int]float64 {
	m := map[int]float64{}
	for _, b := range in {
		m[b.Band] = b.Intensity
	}
	return m
}

func peaksToMap(in []model.DyePeak) map[int]float64 {
	m := map[int]float64{}
	for _, p := range in {
		m[p.Wavelength] = p.Intensity
	}
	return m
}

func layersToMap(in []model.RepairLayer) map[int]model.RepairLayer {
	m := map[int]model.RepairLayer{}
	for _, l := range in {
		m[l.LayerIndex] = l
	}
	return m
}

func unionKeys[K comparable, V any](a, b map[K]V) []K {
	seen := map[K]bool{}
	var keys []K
	for k := range a {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	return keys
}

func toVec(ma, mb map[int]float64) ([]float64, []float64) {
	keys := unionKeys(ma, mb)
	va := make([]float64, len(keys))
	vb := make([]float64, len(keys))
	for i, k := range keys {
		va[i] = ma[k]
		vb[i] = mb[k]
	}
	return va, vb
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return clamp01(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
