// Package spectrum 提供文献样本检测谱（纤维谱/染料峰/修补层序）的校验与规整逻辑。
// 纯函数，无持久化依赖；service 在写入前调用 Validate。
package spectrum

import (
	"fmt"

	"task215-fiberlineage/internal/model"
)

// Validate 校验检测谱是否可进入比对：
//   - 纤维谱非空且每个强度落在 [0,1]，波段号唯一；
//   - 染料峰波长唯一、强度 [0,1]；
//   - 修补层序号唯一、材料非空、厚度 [0,1]；
//   - 单位必须已知（unit_known 为 true，未知则拒绝比对）。
func Validate(det *model.SampleDetection) error {
	if det == nil {
		return fmt.Errorf("%w: nil detection", model.ErrInvalidInput)
	}
	if !det.UnitKnown {
		return fmt.Errorf("%w: detection unit unknown, cannot compare", model.ErrInvalidInput)
	}
	if len(det.FiberSpectrum) == 0 {
		return fmt.Errorf("%w: fiber spectrum empty", model.ErrInvalidInput)
	}
	seenBand := map[int]bool{}
	for _, fb := range det.FiberSpectrum {
		if fb.Band < 0 {
			return fmt.Errorf("%w: negative band %d", model.ErrInvalidInput, fb.Band)
		}
		if seenBand[fb.Band] {
			return fmt.Errorf("%w: duplicate fiber band %d", model.ErrInvalidInput, fb.Band)
		}
		seenBand[fb.Band] = true
		if fb.Intensity < 0 || fb.Intensity > 1 {
			return fmt.Errorf("%w: fiber intensity out of [0,1] at band %d", model.ErrInvalidInput, fb.Band)
		}
	}
	seenW := map[int]bool{}
	for _, dp := range det.DyePeaks {
		if dp.Wavelength <= 0 {
			return fmt.Errorf("%w: non-positive dye wavelength", model.ErrInvalidInput)
		}
		if seenW[dp.Wavelength] {
			return fmt.Errorf("%w: duplicate dye wavelength %d", model.ErrInvalidInput, dp.Wavelength)
		}
		seenW[dp.Wavelength] = true
		if dp.Intensity < 0 || dp.Intensity > 1 {
			return fmt.Errorf("%w: dye intensity out of [0,1]", model.ErrInvalidInput)
		}
	}
	seenLayer := map[int]bool{}
	for _, rl := range det.RepairLayers {
		if rl.LayerIndex < 0 {
			return fmt.Errorf("%w: negative repair layer index", model.ErrInvalidInput)
		}
		if seenLayer[rl.LayerIndex] {
			return fmt.Errorf("%w: duplicate repair layer %d", model.ErrInvalidInput, rl.LayerIndex)
		}
		seenLayer[rl.LayerIndex] = true
		if rl.Material == "" {
			return fmt.Errorf("%w: empty repair material", model.ErrInvalidInput)
		}
		if rl.Thickness < 0 || rl.Thickness > 1 {
			return fmt.Errorf("%w: repair thickness out of [0,1]", model.ErrInvalidInput)
		}
	}
	return nil
}

// NormalizeIntensities 将各类强度线性缩放到 [0,1]，避免量纲差异影响相似度。
// 不做原地修改，返回新的检测谱。
func NormalizeIntensities(det model.SampleDetection) model.SampleDetection {
	out := det
	out.FiberSpectrum = scaleBands(det.FiberSpectrum)
	out.DyePeaks = scalePeaks(det.DyePeaks)
	out.RepairLayers = scaleLayers(det.RepairLayers)
	return out
}

func scaleBands(in []model.FiberBand) []model.FiberBand {
	out := make([]model.FiberBand, len(in))
	copy(out, in)
	max := maxIntensityBand(in)
	if max > 0 {
		for i := range out {
			out[i].Intensity = clamp01(out[i].Intensity / max)
		}
	}
	return out
}

func scalePeaks(in []model.DyePeak) []model.DyePeak {
	out := make([]model.DyePeak, len(in))
	copy(out, in)
	max := 0.0
	for _, p := range in {
		if p.Intensity > max {
			max = p.Intensity
		}
	}
	if max > 0 {
		for i := range out {
			out[i].Intensity = clamp01(out[i].Intensity / max)
		}
	}
	return out
}

func scaleLayers(in []model.RepairLayer) []model.RepairLayer {
	out := make([]model.RepairLayer, len(in))
	copy(out, in)
	max := 0.0
	for _, l := range in {
		if l.Thickness > max {
			max = l.Thickness
		}
	}
	if max > 0 {
		for i := range out {
			out[i].Thickness = clamp01(out[i].Thickness / max)
		}
	}
	return out
}

func maxIntensityBand(in []model.FiberBand) float64 {
	m := 0.0
	for _, b := range in {
		if b.Intensity > m {
			m = b.Intensity
		}
	}
	return m
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
