package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Algo2147483647/ray/engine/ray_tracing"
)

type StudioScript struct {
	Includes  []string                          `json:"includes"`
	Materials []map[string]interface{}          `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []map[string]interface{}          `json:"objects"`
	Cameras   []StudioCameraScript              `json:"cameras"`
	Films     []StudioFilmScript                `json:"films"`
	Render    StudioRenderScript                `json:"render"`
	Geometry  map[string]interface{}            `json:"geometry"`
	Renders   []StudioRenderScript              `json:"renders"`
}

type StudioRenderScript struct {
	Integrator         string `json:"integrator"`
	BDPTFallbackPolicy string `json:"bdpt_fallback_policy,omitempty"`
	Dimension          int    `json:"dimension"`
	Samples            int64  `json:"samples"`
	ThreadNum          int    `json:"thread_num"`
	FilmID             string `json:"film_id"`
	SpectrumMode       string `json:"spectrum_mode"`
	WavelengthSamples  int    `json:"wavelength_samples"`
}

const DefaultSampledWavelengthCount = 4

func NormalizeWavelengthSamples(spectrumMode string, wavelengthSamples int) int {
	if spectrumMode == "sampled" && wavelengthSamples <= 1 {
		return DefaultSampledWavelengthCount
	}
	return wavelengthSamples
}

func MergeRenderScripts(base, override StudioRenderScript) StudioRenderScript {
	if override.Integrator != "" {
		base.Integrator = override.Integrator
	}
	if override.BDPTFallbackPolicy != "" {
		base.BDPTFallbackPolicy = override.BDPTFallbackPolicy
	}
	if override.Dimension > 0 {
		base.Dimension = override.Dimension
	}
	if override.Samples > 0 {
		base.Samples = override.Samples
	}
	if override.ThreadNum > 0 {
		base.ThreadNum = override.ThreadNum
	}
	if override.FilmID != "" {
		base.FilmID = override.FilmID
	}
	if override.SpectrumMode != "" {
		base.SpectrumMode = override.SpectrumMode
	}
	if override.WavelengthSamples > 0 {
		base.WavelengthSamples = override.WavelengthSamples
	}
	return base
}

func (r *StudioRenderScript) UnmarshalJSON(data []byte) error {
	type plain StudioRenderScript
	if err := rejectUnknownFields(data, "render", "integrator", "bdpt_fallback_policy", "dimension", "samples", "thread_num", "film_id", "spectrum_mode", "wavelength_samples"); err != nil {
		return err
	}
	if err := json.Unmarshal(data, (*plain)(r)); err != nil {
		return err
	}
	if r.Dimension < 0 || r.Dimension == 1 {
		return fmt.Errorf("render dimension must be 0 or >= 2")
	}
	if r.ThreadNum < 0 {
		return fmt.Errorf("render thread_num must be >= 0")
	}
	if r.Samples < 0 {
		return fmt.Errorf("render samples must be >= 0")
	}
	if _, err := ray_tracing.ParseIntegratorKind(r.Integrator); err != nil {
		return err
	}
	if r.BDPTFallbackPolicy != "" && r.BDPTFallbackPolicy != string(ray_tracing.BDPTFallbackPath) {
		return fmt.Errorf("unsupported bdpt_fallback_policy %q", r.BDPTFallbackPolicy)
	}
	if r.SpectrumMode == "rgb" {
		r.SpectrumMode = "hero_wavelength"
	} else if r.SpectrumMode != "" && r.SpectrumMode != "hero_wavelength" && r.SpectrumMode != "sampled" {
		return fmt.Errorf("unsupported spectrum_mode %q", r.SpectrumMode)
	}
	if r.WavelengthSamples < 0 {
		return fmt.Errorf("render wavelength_samples must be >= 0")
	}
	return nil
}

// StudioFilmScript owns a Film's sampling grid, camera association, and
// output presentation. Render settings select it through film_id.
type StudioFilmScript struct {
	ID           string              `json:"id"`
	CameraID     string              `json:"camera_id"`
	Shape        []int               `json:"shape"`
	OutputImage  string              `json:"output_image"`
	OutputFilm   string              `json:"output_film"`
	ResumeFilm   string              `json:"resume_film"`
	Exposure     float64             `json:"exposure"`
	ToneMapping  string              `json:"tone_mapping"`
	Gamma        float64             `json:"gamma"`
	ColorSpace   string              `json:"color_space"`
	PixelWindows []PixelWindowScript `json:"pixel_windows"`
}

func (f *StudioFilmScript) UnmarshalJSON(data []byte) error {
	type plain StudioFilmScript
	if err := rejectUnknownFields(data, "film", "id", "camera_id", "shape", "output_image", "output_film", "resume_film", "exposure", "tone_mapping", "gamma", "color_space", "pixel_windows"); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(f))
}

type StudioCameraScript struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	Position     []float64   `json:"position"`
	LookAt       []float64   `json:"look_at"`
	Direction    []float64   `json:"direction"`
	Up           []float64   `json:"up"`
	FieldOfView  float64     `json:"field_of_view"`
	FieldOfViews []float64   `json:"field_of_views"`
	Coordinates  [][]float64 `json:"coordinates"`
	AspectRatio  float64     `json:"aspect_ratio"`
	Ortho        bool        `json:"ortho"`
}

func (c *StudioCameraScript) UnmarshalJSON(data []byte) error {
	type plain StudioCameraScript
	if err := rejectUnknownFields(data, "camera", "id", "type", "position", "look_at", "direction", "up", "field_of_view", "field_of_views", "coordinates", "aspect_ratio", "ortho"); err != nil {
		return err
	}
	return json.Unmarshal(data, (*plain)(c))
}

func rejectUnknownFields(data []byte, kind string, allowed ...string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	known := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		known[key] = true
	}
	for key := range raw {
		if !known[key] && !strings.HasPrefix(key, "_") {
			return fmt.Errorf("unsupported %s field %q", kind, key)
		}
	}
	return nil
}

type PixelWindowScript struct {
	Min []int `json:"min"`
	Max []int `json:"max"`
}

type IntermediateScript struct {
	Studio    StudioMetadata                    `json:"_studio"`
	Materials []map[string]interface{}          `json:"materials,omitempty"`
	Media     map[string]map[string]interface{} `json:"media,omitempty"`
	Objects   []map[string]interface{}          `json:"objects,omitempty"`
	Cameras   []EngineCameraScript              `json:"cameras,omitempty"`
	Geometry  map[string]interface{}            `json:"geometry,omitempty"`
	Renders   []map[string]interface{}          `json:"renders,omitempty"`
}

type StudioMetadata struct {
	Version     string   `json:"version"`
	Source      []string `json:"source"`
	GeneratedAt string   `json:"generated_at"`
	Dimension   int      `json:"dimension"`
}

type EngineCameraScript struct {
	ID           string           `json:"id,omitempty"`
	Type         string           `json:"type,omitempty"`
	Position     []float64        `json:"position,omitempty"`
	FieldOfViews []float64        `json:"field_of_views,omitempty"`
	Coordinates  [][]float64      `json:"coordinates,omitempty"`
	Ortho        bool             `json:"ortho,omitempty"`
	Film         EngineFilmScript `json:"film"`
}

type EngineFilmScript struct {
	Shape        []int               `json:"shape"`
	OutputFilm   string              `json:"output_film,omitempty"`
	PixelWindows []PixelWindowScript `json:"pixel_windows,omitempty"`
}
