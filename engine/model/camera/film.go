package camera

import (
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
)

// Film is a scene-linear spectral image. It deliberately carries no observer,
// tristimulus working space, display transform, or image-encoding policy.
type Film struct {
	Shape         []int                   `json:"shape"`
	OutputFilm    string                  `json:"output_film,omitempty"`
	PixelWindows  []PixelWindow           `json:"pixel_windows,omitempty"`
	Samples       int64                   `json:"samples"`
	SpectralBins  []maths.Tensor[float64] `json:"spectral_bins"`
	SpectralMinNM float64                 `json:"spectral_min_nm"`
	SpectralMaxNM float64                 `json:"spectral_max_nm"`
}

type SpectralSample struct {
	WavelengthNM float64
	Value        float64
}

func NewFilm(shape ...int) *Film {
	return (&Film{}).Init(shape...)
}

func (f *Film) Init(shape ...int) *Film {
	if f == nil {
		return nil
	}
	f.Shape = append(f.Shape[:0], shape...)
	f.Samples = 0
	f.SpectralBins = nil
	f.SpectralMinNM = 0
	f.SpectralMaxNM = 0
	return f
}

// Reset clears accumulated radiance while preserving the Film configuration.
func (f *Film) Reset() {
	if f == nil {
		return
	}
	f.Samples = 0
	f.SpectralBins = nil
	f.SpectralMinNM = 0
	f.SpectralMaxNM = 0
}

func (f *Film) ElementCount() int {
	if f == nil || len(f.Shape) == 0 {
		return 0
	}
	count := 1
	for _, extent := range f.Shape {
		if extent <= 0 {
			return 0
		}
		count *= extent
	}
	return count
}

func (f *Film) InitSpectralBins(count int, minNM, maxNM float64) {
	if f == nil || count <= 0 || f.ElementCount() == 0 || minNM <= 0 || maxNM <= minNM {
		if f != nil {
			f.SpectralBins = nil
			f.SpectralMinNM = 0
			f.SpectralMaxNM = 0
		}
		return
	}
	f.SpectralBins = make([]maths.Tensor[float64], count)
	for i := range f.SpectralBins {
		f.SpectralBins[i] = *maths.NewTensor[float64](append([]int(nil), f.Shape...))
	}
	f.SpectralMinNM = minNM
	f.SpectralMaxNM = maxNM
}

func (f *Film) HasSpectralBins() bool {
	return f != nil && len(f.SpectralBins) > 0 && f.SpectralMaxNM > f.SpectralMinNM
}

func (f *Film) RecordSpectralSample(pixel int, wavelengthNM, value float64) {
	if !f.HasSpectralBins() || pixel < 0 || pixel >= f.ElementCount() ||
		math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	bin := f.SpectralBinIndex(wavelengthNM)
	if bin < 0 {
		return
	}
	f.SpectralBins[bin].Data[pixel] += value
}

func (f *Film) SpectralBinCenterNM(bin int) float64 {
	if !f.HasSpectralBins() || bin < 0 || bin >= len(f.SpectralBins) {
		return 0
	}
	width := (f.SpectralMaxNM - f.SpectralMinNM) / float64(len(f.SpectralBins))
	return f.SpectralMinNM + (float64(bin)+0.5)*width
}

func (f *Film) SpectralBinIndex(wavelengthNM float64) int {
	if !f.HasSpectralBins() || wavelengthNM < f.SpectralMinNM || wavelengthNM >= f.SpectralMaxNM {
		return -1
	}
	index := int((wavelengthNM - f.SpectralMinNM) /
		(f.SpectralMaxNM - f.SpectralMinNM) * float64(len(f.SpectralBins)))
	if index < 0 || index >= len(f.SpectralBins) {
		return -1
	}
	return index
}

type PixelWindow struct {
	Min []int `json:"min"`
	Max []int `json:"max"`
}

func NormalizePixelWindows(windows []PixelWindow, shape []int) ([]PixelWindow, error) {
	if len(windows) == 0 {
		return nil, nil
	}
	if len(shape) == 0 {
		return nil, fmt.Errorf("pixel windows require a non-empty film shape")
	}

	normalized := make([]PixelWindow, len(windows))
	for i, window := range windows {
		if len(window.Min) == 0 || len(window.Max) == 0 {
			return nil, fmt.Errorf("pixel_windows[%d] requires min and max", i)
		}
		if len(window.Min) != len(window.Max) {
			return nil, fmt.Errorf("pixel_windows[%d] min and max must have the same dimension count", i)
		}
		if len(window.Min) > len(shape) {
			return nil, fmt.Errorf("pixel_windows[%d] has %d dimensions, film has %d", i, len(window.Min), len(shape))
		}

		min := make([]int, len(shape))
		max := make([]int, len(shape))
		copy(min, window.Min)
		copy(max, window.Max)
		for dim := len(window.Min); dim < len(shape); dim++ {
			min[dim] = 0
			max[dim] = shape[dim]
		}

		for dim := range shape {
			if min[dim] < 0 || max[dim] < 0 {
				return nil, fmt.Errorf("pixel_windows[%d] dimension %d must be non-negative", i, dim)
			}
			if min[dim] >= max[dim] {
				return nil, fmt.Errorf("pixel_windows[%d] dimension %d requires min < max", i, dim)
			}
			if max[dim] > shape[dim] {
				return nil, fmt.Errorf("pixel_windows[%d] dimension %d max %d exceeds film extent %d", i, dim, max[dim], shape[dim])
			}
		}

		normalized[i] = PixelWindow{Min: min, Max: max}
	}
	return normalized, nil
}
