package ray_tracing

import (
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"runtime"
	"sync"

	"github.com/Algo2147483647/ray/engine/model/material/core"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

type Handler struct {
	MaxRayLevel       int64             `json:"max_ray_level"`
	ThreadNum         int               `json:"thread_num"`
	BlockCols         int               `json:"block_cols"`
	BlockRows         int               `json:"block_rows"`
	SpectrumMode      core.SpectrumMode `json:"spectrum_mode"`
	WavelengthSamples int               `json:"wavelength_samples"`
	RayPool           sync.Pool         `json:"ray_pool"`
}

func NewHandler() *Handler {
	return &Handler{
		MaxRayLevel:       6,
		ThreadNum:         runtime.NumCPU(),
		BlockCols:         8,
		BlockRows:         8,
		SpectrumMode:      core.SpectrumSpectral,
		WavelengthSamples: 1,
		RayPool: sync.Pool{
			New: func() interface{} {
				return &renderray.Ray{
					Origin:    mat.NewVecDense(utils.Dimension, nil),
					Direction: mat.NewVecDense(utils.Dimension, nil),
					Color:     mat.NewVecDense(3, nil),
				}
			},
		},
	}
}

func (h *Handler) EffectiveSampleCount(cameraSamples int64) int64 {
	if cameraSamples <= 0 {
		return 0
	}
	if h.SpectrumMode != core.SpectrumRGBAndSpectral {
		return cameraSamples
	}
	wavelengthSamples := h.WavelengthSamples
	if wavelengthSamples <= 0 {
		wavelengthSamples = 4
	}
	return cameraSamples * int64(wavelengthSamples)
}
