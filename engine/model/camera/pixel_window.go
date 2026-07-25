package camera

import "fmt"

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
