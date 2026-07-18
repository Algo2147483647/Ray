package parser

import (
	"encoding/json"
	"fmt"
)

type CameraScript struct {
	ID           string      `json:"id"`             // Unique camera identifier.
	Type         string      `json:"type"`           // Camera model or script type.
	Position     []float64   `json:"position"`       // Camera origin in scene space.
	LookAt       []float64   `json:"look_at"`        // Target point the camera faces.
	Direction    []float64   `json:"direction"`      // Forward viewing direction.
	Up           []float64   `json:"up"`             // Up vector defining camera roll.
	Widths       []int       `json:"widths"`         // Per-frame image widths.
	FieldOfView  float64     `json:"field_of_view"`  // Vertical field of view in degrees.
	FieldOfViews []float64   `json:"field_of_views"` // Per-frame field-of-view values.
	Coordinates  [][]float64 `json:"coordinates"`    // Camera path or sampled positions.
	AspectRatio  float64     `json:"aspect_ratio"`   // Image width-to-height ratio.
	Ortho        bool        `json:"ortho"`          // Enables orthographic projection.
}

func (c *CameraScript) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw["width"] != nil {
		return fmt.Errorf(`camera field "width" has been removed; set render.width instead`)
	}
	if raw["height"] != nil {
		return fmt.Errorf(`camera field "height" has been removed; set render.height instead`)
	}

	type cameraScript CameraScript
	var decoded cameraScript
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*c = CameraScript(decoded)
	return nil
}

type RenderScript struct {
	Dimension         int     `json:"dimension"`
	Samples           int64   `json:"samples"`
	ThreadNum         int     `json:"thread_num"`
	CameraIndex       int     `json:"camera_index"`
	CameraIndexSet    bool    `json:"-"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	OutputImage       string  `json:"output_image"`
	OutputFilm        string  `json:"output_film"`
	ResumeFilm        string  `json:"resume_film"`
	Exposure          float64 `json:"exposure"`
	ToneMapping       string  `json:"tone_mapping"`
	Gamma             float64 `json:"gamma"`
	SpectrumMode      string  `json:"spectrum_mode"`
	WavelengthSamples int     `json:"wavelength_samples"`
	ColorSpace        string  `json:"color_space"`
	FilmColorSpace    string  `json:"working_space"`
}

func (r *RenderScript) UnmarshalJSON(data []byte) error {
	type renderScript RenderScript
	var decoded renderScript
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded.CameraIndexSet = raw["camera_index"] != nil

	*r = RenderScript(decoded)
	return nil
}

type GeometryScript struct {
	Type   string  `json:"type"`    // "euclidean" | "klein" | "spherical"
	MaxArc float64 `json:"max_arc"` // total geodesic budget per ray; 0 ⇒ defaults (∞ for klein/euclidean, 2π for spherical)
}

type Script struct {
	Includes  []string                          `json:"includes"`
	Materials []map[string]interface{}          `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []map[string]interface{}          `json:"objects"`
	Cameras   []CameraScript                    `json:"cameras"`
	Render    RenderScript                      `json:"render"`
	Geometry  *GeometryScript                   `json:"geometry"`
	Renders   []RenderScript                    `json:"renders"`
}
