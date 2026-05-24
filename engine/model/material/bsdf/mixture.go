package bsdf

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"math"
)

type WeightedBxDF struct {
	Weight float64
	BxDF   bxdf.BxDF
}

type WeightedMixture struct {
	Components []WeightedBxDF
}

func NewWeightedMixture(components ...WeightedBxDF) WeightedMixture {
	return WeightedMixture{Components: components}
}

func (m WeightedMixture) Eval(ctx bxdf.ShadingContext, wi, wo maths.Direction) optics.Spectrum {
	total := m.totalWeight()
	if total <= 0 {
		return optics.Spectrum{}
	}

	result := optics.Spectrum{}
	for _, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}
		result = result.Add(component.BxDF.Eval(ctx, wi, wo).MulScalar(component.Weight / total))
	}
	return result
}

func (m WeightedMixture) Sample(ctx bxdf.ShadingContext, wo maths.Direction, u maths.Sample2D) bxdf.BxDFSample {
	total := m.totalWeight()
	if total <= 0 {
		return bxdf.BxDFSample{}
	}

	index, remappedU, ok := m.selectComponent(u.U, total)
	if !ok {
		return bxdf.BxDFSample{}
	}

	selected := m.Components[index].BxDF
	sample := selected.Sample(ctx, wo, maths.Sample2D{U: remappedU, V: u.V})
	if sample.PDF == 0 {
		return sample
	}

	selectedWeight := m.Components[index].Weight / total
	if isDeltaSample(sample) {
		sample.F = sample.F.MulScalar(selectedWeight)
		sample.PDF *= selectedWeight
		return sample
	}

	sample.F = m.Eval(ctx, sample.Wi, wo)
	sample.PDF = m.PDF(ctx, sample.Wi, wo)
	return sample
}

func (m WeightedMixture) PDF(ctx bxdf.ShadingContext, wi, wo maths.Direction) float64 {
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

func (m WeightedMixture) AlbedoBound(ctx bxdf.ShadingContext) optics.Spectrum {
	total := m.totalWeight()
	if total <= 0 {
		return optics.Spectrum{}
	}

	result := optics.Spectrum{}
	for _, component := range m.Components {
		if component.BxDF == nil || component.Weight <= 0 {
			continue
		}
		result = result.Add(component.BxDF.AlbedoBound(ctx).MulScalar(component.Weight / total))
	}
	return result
}

func (m WeightedMixture) RoughnessInfo(ctx bxdf.ShadingContext) bxdf.RoughnessInfo {
	info := bxdf.RoughnessInfo{IsDelta: true}
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

func (m WeightedMixture) DeltaFlags() bxdf.DeltaFlags {
	flags := bxdf.DeltaNone
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

func isDeltaSample(sample bxdf.BxDFSample) bool {
	return sample.Flags&(bxdf.DeltaReflection|bxdf.DeltaTransmission) != 0
}
