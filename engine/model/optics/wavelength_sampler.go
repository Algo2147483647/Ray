package optics

import (
	"math"
)

type WavelengthSample struct {
	LambdaNM float64
	PDF      float64
}

type WavelengthSampler interface {
	Sample(u float64) WavelengthSample
}

type UniformWavelengthSampler struct {
	MinNM float64
	MaxNM float64
}

func NewUniformWavelengthSampler() UniformWavelengthSampler {
	return UniformWavelengthSampler{
		MinNM: WavelengthMin,
		MaxNM: WavelengthMax,
	}
}

type WavelengthWeightFunc func(wavelengthNM float64) float64

type WeightedWavelengthSampler struct {
	MinNM   float64
	MaxNM   float64
	Bins    int
	centers []float64
	pdfs    []float64
	cdf     []float64
}

func NewWeightedWavelengthSampler(minNM, maxNM float64, bins int, weight WavelengthWeightFunc) WeightedWavelengthSampler {
	if minNM <= 0 {
		minNM = WavelengthMin
	}
	if maxNM <= minNM {
		maxNM = WavelengthMax
	}
	if bins <= 0 {
		bins = 64
	}
	if weight == nil {
		weight = func(float64) float64 { return 1 }
	}

	width := (maxNM - minNM) / float64(bins)
	centers := make([]float64, bins)
	weights := make([]float64, bins)
	total := 0.0
	for i := 0; i < bins; i++ {
		wavelength := minNM + (float64(i)+0.5)*width
		w := math.Max(0, weight(wavelength))
		centers[i] = wavelength
		weights[i] = w
		total += w * width
	}
	if total <= 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		return NewWeightedWavelengthSampler(minNM, maxNM, bins, nil)
	}

	pdfs := make([]float64, bins)
	cdf := make([]float64, bins)
	accum := 0.0
	for i, w := range weights {
		pdfs[i] = w / total
		accum += pdfs[i] * width
		cdf[i] = accum
	}
	cdf[len(cdf)-1] = 1

	return WeightedWavelengthSampler{
		MinNM:   minNM,
		MaxNM:   maxNM,
		Bins:    bins,
		centers: centers,
		pdfs:    pdfs,
		cdf:     cdf,
	}
}

func NewCIESensitivityWavelengthSampler(bins int) WeightedWavelengthSampler {
	return NewWeightedWavelengthSampler(WavelengthMin, WavelengthMax, bins, func(wavelengthNM float64) float64 {
		xyz := WavelengthToXYZ(wavelengthNM)
		return xyz.AtVec(1)
	})
}

func NewRGBImportanceWavelengthSampler(rgb Spectrum, bins int) WeightedWavelengthSampler {
	return NewWeightedWavelengthSampler(WavelengthMin, WavelengthMax, bins, func(wavelengthNM float64) float64 {
		return rgb.RGBPowerAtWavelength(wavelengthNM)
	})
}

func NewCompositeWavelengthSampler(bins int, weights ...WavelengthWeightFunc) WeightedWavelengthSampler {
	return NewWeightedWavelengthSampler(WavelengthMin, WavelengthMax, bins, func(wavelengthNM float64) float64 {
		total := 0.0
		for _, weight := range weights {
			if weight != nil {
				total += math.Max(0, weight(wavelengthNM))
			}
		}
		return total
	})
}

func (s WeightedWavelengthSampler) Sample(u float64) WavelengthSample {
	if len(s.cdf) == 0 || len(s.pdfs) == 0 || len(s.centers) == 0 {
		return NewUniformWavelengthSampler().Sample(u)
	}
	u = math.Max(1e-6, math.Min(1-1e-6, u))
	idx := 0
	for idx < len(s.cdf)-1 && u > s.cdf[idx] {
		idx++
	}
	minNM := s.MinNM
	maxNM := s.MaxNM
	if minNM <= 0 {
		minNM = WavelengthMin
	}
	if maxNM <= minNM {
		maxNM = WavelengthMax
	}
	width := (maxNM - minNM) / float64(len(s.pdfs))
	prevCDF := 0.0
	if idx > 0 {
		prevCDF = s.cdf[idx-1]
	}
	binMass := s.cdf[idx] - prevCDF
	localU := 0.5
	if binMass > 0 {
		localU = (u - prevCDF) / binMass
	}
	localU = math.Max(1e-6, math.Min(1-1e-6, localU))
	return WavelengthSample{
		LambdaNM: minNM + (float64(idx)+localU)*width,
		PDF:      s.pdfs[idx],
	}
}

func (s UniformWavelengthSampler) Sample(u float64) WavelengthSample {
	minNM := s.MinNM
	maxNM := s.MaxNM
	if minNM <= 0 {
		minNM = WavelengthMin
	}
	if maxNM <= minNM {
		maxNM = WavelengthMax
	}

	u = math.Max(1e-6, math.Min(1-1e-6, u))
	return WavelengthSample{
		LambdaNM: minNM + u*(maxNM-minNM),
		PDF:      1 / (maxNM - minNM),
	}
}
