package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
	"sync"
)

type Handler struct {
	IntegratorKind       IntegratorKind           `json:"integrator"`
	MaxRayLevel          int64                    `json:"max_ray_level"`
	RussianRouletteDepth int64                    `json:"russian_roulette_depth"`
	MaxArc               float64                  `json:"max_arc"` // total geodesic distance budget per ray (0 ⇒ unbounded)
	Space                geometry.SceneSpace      `json:"-"`
	ThreadNum            int                      `json:"thread_num"`
	BlockCols            int                      `json:"block_cols"`
	BlockRows            int                      `json:"block_rows"`
	SpectrumMode         optics.SpectrumMode      `json:"spectrum_mode"`
	WavelengthSamples    int                      `json:"wavelength_samples"`
	WavelengthSampler    optics.WavelengthSampler `json:"-"`
	RayPool              sync.Pool                `json:"ray_pool"`
}

func NewHandler(space geometry.SceneSpace) *Handler {
	space = geometry.NewSceneSpace(space.Geometry, space.Dimension)
	return &Handler{
		MaxRayLevel:          64,
		RussianRouletteDepth: 3,
		MaxArc:               0, // 0 means unbounded; set by scene factory for spherical scenes.
		BlockCols:            8,
		BlockRows:            8,
		Space:                space,
		RayPool: sync.Pool{
			New: func() interface{} {
				return &optics.Ray{
					Origin:    mat.NewVecDense(space.Dimension, nil),
					Direction: mat.NewVecDense(space.Dimension, nil),
					Space:     space,
				}
			},
		},
	}
}

func (h *Handler) wavelengthSampler() optics.WavelengthSampler {
	if h.WavelengthSampler != nil {
		return h.WavelengthSampler
	}
	return optics.NewUniformWavelengthSampler()
}

func (h *Handler) EffectiveSampleCount(cameraSamples int64) int64 {
	if cameraSamples <= 0 {
		return 0
	}
	if h.SpectrumMode != optics.SpectrumModeSampledWavelengths {
		return cameraSamples
	}
	return cameraSamples * int64(h.WavelengthSamples)
}
