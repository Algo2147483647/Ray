package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/optics"
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
		MinNM: optics.WavelengthMin,
		MaxNM: optics.WavelengthMax,
	}
}

func (s UniformWavelengthSampler) Sample(u float64) WavelengthSample {
	minNM := s.MinNM
	maxNM := s.MaxNM
	if minNM <= 0 {
		minNM = optics.WavelengthMin
	}
	if maxNM <= minNM {
		maxNM = optics.WavelengthMax
	}

	u = math.Max(1e-6, math.Min(1-1e-6, u))
	return WavelengthSample{
		LambdaNM: minNM + u*(maxNM-minNM),
		PDF:      1 / (maxNM - minNM),
	}
}
