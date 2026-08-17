package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"runtime"
	"sync"
)

type Handler struct {
	IntegratorKind          IntegratorKind           `json:"integrator"`
	MaxRayLevel             int64                    `json:"max_ray_level"`
	RussianRouletteDepth    int64                    `json:"russian_roulette_depth"`
	MaxArc                  float64                  `json:"max_arc"` // total geodesic distance budget per ray (0 ⇒ unbounded)
	SceneGeometry           geometry.Geometry        `json:"-"`
	ThreadNum               int                      `json:"thread_num"`
	BlockCols               int                      `json:"block_cols"`
	BlockRows               int                      `json:"block_rows"`
	SpectrumMode            optics.SpectrumMode      `json:"spectrum_mode"`
	WavelengthSamples       int                      `json:"wavelength_samples"`
	WavelengthSampler       optics.WavelengthSampler `json:"-"`
	BDPTFallbackPolicy      BDPTFallbackPolicy       `json:"bdpt_fallback_policy,omitempty"`
	LastRequestedIntegrator IntegratorKind           `json:"-"`
	LastEffectiveIntegrator IntegratorKind           `json:"-"`
	LastFallbackReason      string                   `json:"-"`
	RayPool                 sync.Pool                `json:"ray_pool"`
}

func NewHandler() *Handler {
	return &Handler{
		IntegratorKind:       IntegratorPathTracing,
		MaxRayLevel:          64,
		RussianRouletteDepth: 3,
		MaxArc:               0, // 0 means unbounded; set by scene factory for spherical scenes.
		ThreadNum:            runtime.NumCPU(),
		BlockCols:            8,
		BlockRows:            8,
		SpectrumMode:         optics.SpectrumModeHeroWavelength,
		WavelengthSamples:    1,
		RayPool: sync.Pool{
			New: func() interface{} {
				return &optics.Ray{
					Origin:    mat.NewVecDense(utils.Dimension, nil),
					Direction: mat.NewVecDense(utils.Dimension, nil),
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
	wavelengthSamples := h.WavelengthSamples
	if wavelengthSamples <= 0 {
		wavelengthSamples = 4
	}
	return cameraSamples * int64(wavelengthSamples)
}
