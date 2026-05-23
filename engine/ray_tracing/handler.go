package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
	"runtime"
	"sync"
)

type Handler struct {
	MaxRayLevel          int64                    `json:"max_ray_level"`
	RussianRouletteDepth int64                    `json:"russian_roulette_depth"`
	ThreadNum            int                      `json:"thread_num"`
	BlockCols            int                      `json:"block_cols"`
	BlockRows            int                      `json:"block_rows"`
	SpectrumMode         optics.SpectrumMode      `json:"spectrum_mode"`
	WavelengthSamples    int                      `json:"wavelength_samples"`
	FilmColorSpace       camera.FilmColorSpace    `json:"working_space"`
	WavelengthSampler    optics.WavelengthSampler `json:"-"`
	RayPool              sync.Pool                `json:"ray_pool"`
}

func NewHandler() *Handler {
	return &Handler{
		MaxRayLevel:          64,
		RussianRouletteDepth: 3,
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
