package film

import "github.com/Algo2147483647/ray/engine/utils"

// Spec is the complete authored configuration of a render job's Film.
// Accumulation state exists only on the resolved runtime Film.
type Spec struct {
	Shape            []int         `json:"shape"`
	PixelWindows     []PixelWindow `json:"pixel_windows,omitempty"`
	SpectralBinCount int           `json:"spectral_bin_count,omitempty"`
}

func (s *Spec) UnmarshalJSON(data []byte) error {
	type plain Spec
	return utils.DecodeStrictJSON(
		data, "film", (*plain)(s),
		"shape", "pixel_windows", "spectral_bin_count",
	)
}
