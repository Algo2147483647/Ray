package parser

import (
	"encoding/json"
	"fmt"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

type Script struct {
	Dimension int                               `json:"dimension"`
	Materials []map[string]interface{}          `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []map[string]interface{}          `json:"objects"`
	Cameras   []CameraScript                    `json:"cameras"`
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
	Film         *modelcamera.Film      `json:"film"`           // Film owned by this camera.
}

type RenderScript struct {
	Integrator        string `json:"integrator"`
	LegacyDimension   int    `json:"dimension,omitempty"` // deprecated: use Script.Dimension
	Samples           int64  `json:"samples"`
	ThreadNum         int    `json:"thread_num"`
	CameraID          string `json:"camera_id"`
	SpectrumMode      string `json:"spectrum_mode"`
	WavelengthSamples int    `json:"wavelength_samples"`
}

func (r *RenderScript) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if _, exists := raw["bdpt_fallback_policy"]; exists {
		return fmt.Errorf("render field %q was removed; choose integrator explicitly", "bdpt_fallback_policy")
	}
	type plain RenderScript
	return json.Unmarshal(data, (*plain)(r))
}

type GeometryScript struct {
	Type   string  `json:"type"`    // "euclidean" | "klein" | "spherical"
	MaxArc float64 `json:"max_arc"` // total geodesic budget per ray; 0 ⇒ defaults (∞ for klein/euclidean, 2π for spherical)
}
