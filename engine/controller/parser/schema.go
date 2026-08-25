package parser

import (
	"encoding/json"
	"fmt"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/utils"
)

type Script struct {
	Dimension int                   `json:"dimension"`
	Materials []MaterialSpec        `json:"materials"`
	Media     map[string]MediumSpec `json:"media"`
	Objects   []ObjectSpec          `json:"objects"`
	Cameras   []CameraScript        `json:"cameras"`
	Geometry  *GeometryScript       `json:"geometry"`
	Renders   []RenderScript        `json:"renders"`
}

func (s *Script) UnmarshalJSON(data []byte) error {
	type plain Script
	var decoded plain
	if err := utils.DecodeStrictJSON(
		data,
		"script",
		&decoded,
		"dimension", "materials", "media", "objects", "cameras", "geometry", "renders",
	); err != nil {
		return err
	}
	*s = Script(decoded)
	return nil
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
	type plain CameraScript
	return utils.DecodeStrictJSON(
		data,
		"camera",
		(*plain)(c),
		"id", "type", "position", "field_of_views", "coordinates", "ortho",
	)
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
	type plain RenderScript
	return utils.DecodeStrictJSON(
		data,
		"render",
		(*plain)(r),
		"integrator", "samples", "thread_num", "camera_id", "wavelength_samples", "film", "output",
	)
}

type GeometryScript struct {
	Type   string  `json:"type"`    // "euclidean" | "klein" | "spherical"
	MaxArc float64 `json:"max_arc"` // total geodesic budget per ray; 0 ⇒ defaults (∞ for klein/euclidean, 2π for spherical)
}

func (g *GeometryScript) UnmarshalJSON(data []byte) error {
	type plain GeometryScript
	return utils.DecodeStrictJSON(data, "geometry", (*plain)(g), "type", "max_arc")
}

type MediumKind string

const MediumHomogeneous MediumKind = "homogeneous"

type MediumSpec struct {
	Type   MediumKind      `json:"type,omitempty"`
	IOR    *IORSpec        `json:"ior,omitempty"`
	SigmaA json.RawMessage `json:"sigma_a,omitempty"`
	SigmaS json.RawMessage `json:"sigma_s,omitempty"`
}

func (s *MediumSpec) UnmarshalJSON(data []byte) error {
	type plain MediumSpec
	if err := utils.DecodeStrictJSON(
		data,
		"medium",
		(*plain)(s),
		"type", "ior", "sigma_a", "sigma_s",
	); err != nil {
		return err
	}
	if s.Type != "" && s.Type != MediumHomogeneous {
		return fmt.Errorf("unsupported medium type %q", s.Type)
	}
	return nil
}
