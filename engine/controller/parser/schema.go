package parser

import (
	"encoding/json"
	"fmt"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

type Script struct {
	Dimension int                               `json:"dimension"`
	Materials []MaterialSpec                    `json:"materials"`
	Media     map[string]map[string]interface{} `json:"media"`
	Objects   []ObjectSpec                      `json:"objects"`
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
}

func (c *CameraScript) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	allowed := map[string]bool{
		"id": true, "type": true, "position": true, "field_of_views": true,
		"coordinates": true, "ortho": true,
	}
	for field := range raw {
		if !allowed[field] {
			return fmt.Errorf("unsupported camera field %q", field)
		}
	}
	type plain CameraScript
	return json.Unmarshal(data, (*plain)(c))
}

type RenderScript struct {
	Integrator        string                `json:"integrator"`
	Samples           int64                 `json:"samples"`
	ThreadNum         int                   `json:"thread_num"`
	CameraID          string                `json:"camera_id"`
	WavelengthSamples int                   `json:"wavelength_samples"`
	Film              *modelcamera.FilmSpec `json:"film"`
	Output            string                `json:"output"`
}

func (r *RenderScript) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	allowed := map[string]bool{
		"integrator": true, "samples": true, "thread_num": true,
		"camera_id": true, "wavelength_samples": true,
		"film": true, "output": true,
	}
	for field := range raw {
		if !allowed[field] {
			return fmt.Errorf("unsupported render field %q", field)
		}
	}
	type plain RenderScript
	return json.Unmarshal(data, (*plain)(r))
}

type GeometryScript struct {
	Type   string  `json:"type"`    // "euclidean" | "klein" | "spherical"
	MaxArc float64 `json:"max_arc"` // total geodesic budget per ray; 0 ⇒ defaults (∞ for klein/euclidean, 2π for spherical)
}
