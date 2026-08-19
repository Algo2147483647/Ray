package emission

import (
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// DirectionFlags describes the measure used by an emitted-direction sample.
type DirectionFlags uint32

const (
	DirectionContinuous DirectionFlags = 0
	DirectionDelta      DirectionFlags = 1 << 0
)

// DirectionSample is expressed in the local geometric-normal frame. PDF is a
// density with respect to solid angle; for a delta emitter it is a discrete
// probability mass and DirectionDelta must be set.
type DirectionSample struct {
	Wo    maths.Direction
	Le    optics.Spectrum
	PDF   float64
	Flags DirectionFlags
}

// Emitter is the complete directional contract required by unidirectional and
// bidirectional light-transport algorithms.
type Emitter interface {
	Eval(ctx bxdf.ShadingContext, woLocal maths.Direction) optics.Spectrum
	SampleDirection(ctx bxdf.ShadingContext, u maths.Sample2D) DirectionSample
	PDFDirection(ctx bxdf.ShadingContext, woLocal maths.Direction) float64
	ExitanceEstimate(ctx bxdf.ShadingContext) optics.Spectrum
	DirectionFlags() DirectionFlags
}

// RadianceField supplies the spatial and spectral part of surface emission.
// Angular variation is deliberately handled by AngularDistribution.
type RadianceField interface {
	EvaluateRadiance(ctx bxdf.ShadingContext) optics.Spectrum
}
