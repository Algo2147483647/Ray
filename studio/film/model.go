package film

import (
	"fmt"
	"math"
)

const MaxSpectralBinCount = 4096

// Film is Studio's independent representation of the versioned RAYFILM
// process artifact. It intentionally does not depend on Engine runtime types.
type Film struct {
	Shape         []int
	PixelWindows  []PixelWindow
	Samples       int64
	SpectralBins  []SpectralPlane
	SpectralMinNM float64
	SpectralMaxNM float64
}

type SpectralPlane struct {
	Shape  []int
	Stride []int
	Data   []float64
}

func newSpectralPlane(shape []int) SpectralPlane {
	stride := make([]int, len(shape))
	total := 1
	for index, extent := range shape {
		stride[index] = total
		total *= extent
	}
	return SpectralPlane{Shape: append([]int(nil), shape...), Stride: stride, Data: make([]float64, total)}
}

func (plane *SpectralPlane) GetCoordinates(index int) []int {
	if plane == nil || index < 0 || index >= len(plane.Data) {
		return nil
	}
	actual := index
	coordinates := make([]int, len(plane.Shape))
	for dimension := len(plane.Stride) - 1; dimension >= 0; dimension-- {
		coordinates[dimension] = actual / plane.Stride[dimension]
		actual %= plane.Stride[dimension]
	}
	return coordinates
}

func NewFilm(shape ...int) *Film { return (&Film{}).Init(shape...) }

func (film *Film) Init(shape ...int) *Film {
	if film == nil {
		return nil
	}
	film.Shape = append(film.Shape[:0], shape...)
	film.Samples = 0
	film.SpectralBins = nil
	film.SpectralMinNM = 0
	film.SpectralMaxNM = 0
	return film
}

func (film *Film) ElementCount() int {
	if film == nil || len(film.Shape) == 0 {
		return 0
	}
	count := 1
	for _, extent := range film.Shape {
		if extent <= 0 {
			return 0
		}
		count *= extent
	}
	return count
}

func (film *Film) InitSpectralBins(count int, minNM, maxNM float64) {
	if film == nil {
		return
	}
	if count <= 0 || film.ElementCount() == 0 || minNM <= 0 || maxNM <= minNM {
		film.SpectralBins, film.SpectralMinNM, film.SpectralMaxNM = nil, 0, 0
		return
	}
	film.SpectralBins = make([]SpectralPlane, count)
	for index := range film.SpectralBins {
		film.SpectralBins[index] = newSpectralPlane(film.Shape)
	}
	film.SpectralMinNM, film.SpectralMaxNM = minNM, maxNM
}

func (film *Film) SpectralBinCount() int {
	if film == nil {
		return 0
	}
	return len(film.SpectralBins)
}

func (film *Film) HasSpectralBins() bool {
	return film != nil && len(film.SpectralBins) > 0 && film.SpectralMaxNM > film.SpectralMinNM
}

func (film *Film) SpectralBinCenterNM(bin int) float64 {
	if !film.HasSpectralBins() || bin < 0 || bin >= len(film.SpectralBins) {
		return 0
	}
	width := (film.SpectralMaxNM - film.SpectralMinNM) / float64(len(film.SpectralBins))
	return film.SpectralMinNM + (float64(bin)+0.5)*width
}

type PixelWindow struct{ Min, Max []int }

func NormalizePixelWindows(windows []PixelWindow, shape []int) ([]PixelWindow, error) {
	if len(windows) == 0 {
		return nil, nil
	}
	if len(shape) == 0 {
		return nil, fmt.Errorf("pixel windows require a non-empty film shape")
	}
	result := make([]PixelWindow, len(windows))
	for index, window := range windows {
		if len(window.Min) == 0 || len(window.Max) == 0 {
			return nil, fmt.Errorf("pixel_windows[%d] requires min and max", index)
		}
		if len(window.Min) != len(window.Max) {
			return nil, fmt.Errorf("pixel_windows[%d] min and max must have the same dimension count", index)
		}
		if len(window.Min) > len(shape) {
			return nil, fmt.Errorf("pixel_windows[%d] has %d dimensions, film has %d", index, len(window.Min), len(shape))
		}
		minimum, maximum := make([]int, len(shape)), make([]int, len(shape))
		copy(minimum, window.Min)
		copy(maximum, window.Max)
		for dimension := len(window.Min); dimension < len(shape); dimension++ {
			maximum[dimension] = shape[dimension]
		}
		for dimension := range shape {
			if minimum[dimension] < 0 || maximum[dimension] < 0 {
				return nil, fmt.Errorf("pixel_windows[%d] dimension %d must be non-negative", index, dimension)
			}
			if minimum[dimension] >= maximum[dimension] {
				return nil, fmt.Errorf("pixel_windows[%d] dimension %d requires min < max", index, dimension)
			}
			if maximum[dimension] > shape[dimension] {
				return nil, fmt.Errorf("pixel_windows[%d] dimension %d max %d exceeds film extent %d", index, dimension, maximum[dimension], shape[dimension])
			}
		}
		result[index] = PixelWindow{Min: minimum, Max: maximum}
	}
	return result, nil
}

func validSpectralRange(minNM, maxNM float64) bool {
	return minNM > 0 && maxNM > minNM && !math.IsNaN(minNM) && !math.IsNaN(maxNM) && !math.IsInf(minNM, 0) && !math.IsInf(maxNM, 0)
}
