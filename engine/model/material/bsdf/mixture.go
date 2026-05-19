package bsdf

import (
	"math"

	"github.com/Algo2147483647/ray/engine/model/material/core"
)

type WeightedBxDF struct {
	Weight float64
	BxDF   core.BxDF
}

type WeightedMixture struct {
	Components []WeightedBxDF
}

func NewWeightedMixture(components ...WeightedBxDF) WeightedMixture {
	return WeightedMixture{Components: components}
}

func (m WeightedMixture) Eval(ctx core.ShadingContext, wi, wo core.Direction) core.Spectrum {
	total := m.totalWeight()
	if total <= 0 {
		return core.Spectrum{}
	}

	result := core.Spectrum{}
	for _, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}
		result = result.Add(component.BxDF.Eval(ctx, wi, wo).MulScalar(component.Weight / total))
	}
	return result
}

func (m WeightedMixture) Sample(ctx core.ShadingContext, wo core.Direction, u core.Sample2D) core.BxDFSample {
	total := m.totalWeight()
	if total <= 0 {
		return core.BxDFSample{}
	}

	index, remappedU, ok := m.selectComponent(u.U, total)
	if !ok {
		return core.BxDFSample{}
	}

	selected := m.Components[index].BxDF
	sample := selected.Sample(ctx, wo, core.Sample2D{U: remappedU, V: u.V})
	if sample.PDF == 0 {
		return sample
	}

	sample.F = m.Eval(ctx, sample.Wi, wo)
	sample.PDF = m.PDF(ctx, sample.Wi, wo)
	sample.Flags = m.DeltaFlags()
	return sample
}

func (m WeightedMixture) PDF(ctx core.ShadingContext, wi, wo core.Direction) float64 {
	total := m.totalWeight()
	if total <= 0 {
		return 0
	}

	var pdf float64
	for _, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}
		pdf += component.BxDF.PDF(ctx, wi, wo) * component.Weight / total
	}
	return pdf
}

func (m WeightedMixture) AlbedoBound(ctx core.ShadingContext) core.Spectrum {
	total := m.totalWeight()
	if total <= 0 {
		return core.Spectrum{}
	}

	result := core.Spectrum{}
	for _, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}
		result = result.Add(component.BxDF.AlbedoBound(ctx).MulScalar(component.Weight / total))
	}
	return result
}

func (m WeightedMixture) RoughnessInfo(ctx core.ShadingContext) core.RoughnessInfo {
	info := core.RoughnessInfo{IsDelta: true}
	for _, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}
		componentInfo := component.BxDF.RoughnessInfo(ctx)
		info.IsDelta = info.IsDelta && componentInfo.IsDelta
		info.AlphaX = math.Max(info.AlphaX, componentInfo.AlphaX)
		info.AlphaY = math.Max(info.AlphaY, componentInfo.AlphaY)
	}
	return info
}

func (m WeightedMixture) DeltaFlags() core.DeltaFlags {
	flags := core.DeltaNone
	for _, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}
		flags |= component.BxDF.DeltaFlags()
	}
	return flags
}

func (m WeightedMixture) totalWeight() float64 {
	var total float64
	for _, component := range m.Components {
		if component.BxDF != nil && component.Weight > 0 {
			total += component.Weight
		}
	}
	return total
}

func (m WeightedMixture) selectComponent(u, total float64) (int, float64, bool) {
	target := clamp01Open(u) * total
	var prefix float64
	for i, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}

		next := prefix + component.Weight
		if target < next {
			return i, (target - prefix) / component.Weight, true
		}
		prefix = next
	}
	return 0, 0, false
}

func clamp01Open(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return math.Nextafter(1, 0)
	}
	return v
}
