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
	FieldOfViews []float64              `json:"field_of_views"` // Per-frame field-of-view values.
	Coordinates  [][]float64            `json:"coordinates"`    // Camera path or sampled positions.
	Ortho        bool                   `json:"ortho"`          // Enables orthographic projection.
}

type RenderScript struct {
	Integrator        string                    `json:"integrator"`
	Dimension         int                       `json:"dimension"`
	Samples           int64                     `json:"samples"`
	ThreadNum         int                       `json:"thread_num"`
	CameraIndex       int                       `json:"camera_index"`
	CameraIndexSet    bool                      `json:"-"`
	Width             []int                     `json:"widths"`
	OutputFilm        string                    `json:"output_film"`
	SpectrumMode      string                    `json:"spectrum_mode"`
	WavelengthSamples int                       `json:"wavelength_samples"`
	PixelWindows      []modelcamera.PixelWindow `json:"pixel_windows"`
}

type GeometryScript struct {
	Type   string  `json:"type"`    // "euclidean" | "klein" | "spherical"
	MaxArc float64 `json:"max_arc"` // total geodesic budget per ray; 0 ⇒ defaults (∞ for klein/euclidean, 2π for spherical)
}
