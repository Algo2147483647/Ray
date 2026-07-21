package parser

import (
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

type Script struct {
	Materials []map[string]interface{}          `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []map[string]interface{}          `json:"objects"`
	Cameras   []CameraScript                    `json:"cameras"`
	Render    RenderScript                      `json:"render"`
	Geometry  *GeometryScript                   `json:"geometry"`
	Renders   []RenderScript                    `json:"renders"`
}

type CameraScript struct {
	ID           string                 `json:"id"`             // Unique camera identifier.
	Type         modelcamera.CameraType `json:"type"`           // Camera model type.
	Position     []float64              `json:"position"`       // Camera origin in scene space.
	Direction    []float64              `json:"direction"`      // Forward viewing direction.
	Up           []float64              `json:"up"`             // Up vector defining camera roll.
	Widths       []int                  `json:"widths"`         // Per-frame image widths.
	FieldOfViews []float64              `json:"field_of_views"` // Per-frame field-of-view values.
	Coordinates  [][]float64            `json:"coordinates"`    // Camera path or sampled positions.
	Ortho        bool                   `json:"ortho"`          // Enables orthographic projection.
}

type RenderScript struct {
	Dimension         int     `json:"dimension"`
	Samples           int64   `json:"samples"`
	ThreadNum         int     `json:"thread_num"`
	CameraIndex       int     `json:"camera_index"`
	CameraIndexSet    bool    `json:"-"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	OutputFilm        string  `json:"output_film"`
	Exposure          float64 `json:"exposure"`
	ToneMapping       string  `json:"tone_mapping"`
	Gamma             float64 `json:"gamma"`
	SpectrumMode      string  `json:"spectrum_mode"`
	WavelengthSamples int     `json:"wavelength_samples"`
	ColorSpace        string  `json:"color_space"`
	FilmColorSpace    string  `json:"working_space"`
}

type GeometryScript struct {
	Type   string  `json:"type"`    // "euclidean" | "klein" | "spherical"
	MaxArc float64 `json:"max_arc"` // total geodesic budget per ray; 0 ⇒ defaults (∞ for klein/euclidean, 2π for spherical)
}
