package camera

import (
	"encoding/json"
	"fmt"
)

// FilmSpec is the complete authored configuration of a render target's Film.
// Accumulation state exists only on the resolved runtime Film.
type FilmSpec struct {
	Shape            []int         `json:"shape"`
	PixelWindows     []PixelWindow `json:"pixel_windows,omitempty"`
	SpectralBinCount int           `json:"spectral_bin_count,omitempty"`
}

func (s *FilmSpec) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	allowed := map[string]bool{
		"shape": true, "pixel_windows": true, "spectral_bin_count": true,
	}
	for field := range raw {
		if !allowed[field] {
			return fmt.Errorf("unsupported film field %q", field)
		}
	}
	type plain FilmSpec
	return json.Unmarshal(data, (*plain)(s))
}
