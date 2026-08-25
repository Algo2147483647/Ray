package parser

import (
	"encoding/json"
	"fmt"
)

// ShapeKind is the discriminator for an Engine object definition.
type ShapeKind string

const (
	ShapeCuboid             ShapeKind = "cuboid"
	ShapeHypercuboid        ShapeKind = "hypercuboid"
	ShapeSphere             ShapeKind = "sphere"
	ShapeHypersphere        ShapeKind = "hypersphere"
	ShapeCircle             ShapeKind = "circle"
	ShapeCylinder           ShapeKind = "cylinder"
	ShapeFiniteCylinder     ShapeKind = "finite cylinder"
	ShapeTriangle           ShapeKind = "triangle"
	ShapePlane              ShapeKind = "plane"
	ShapeQuadraticEquation  ShapeKind = "quadratic equation"
	ShapeCubicEquation      ShapeKind = "cubic equation"
	ShapeFourOrderEquation  ShapeKind = "four-order equation"
	ShapeImplicitEquation   ShapeKind = "implicit equation"
	ShapeParametricEquation ShapeKind = "parametric equation"
	ShapeParametricCurve    ShapeKind = "parametric curve"
	ShapePolynomialSurface  ShapeKind = "polynomial surface"
	ShapeKleinBottle        ShapeKind = "klein_bottle"
	ShapeSTL                ShapeKind = "stl"
)

var supportedShapeKinds = map[ShapeKind]bool{
	ShapeCuboid: true, ShapeHypercuboid: true, ShapeSphere: true,
	ShapeHypersphere: true, ShapeCircle: true, ShapeCylinder: true,
	ShapeFiniteCylinder: true, ShapeTriangle: true, ShapePlane: true,
	ShapeQuadraticEquation: true, ShapeCubicEquation: true,
	ShapeFourOrderEquation: true, ShapeImplicitEquation: true,
	ShapeParametricEquation: true, ShapeParametricCurve: true,
	ShapePolynomialSurface: true, ShapeKleinBottle: true, ShapeSTL: true,
}

type BoundsSpec struct {
	PMin []float64 `json:"pmin"`
	PMax []float64 `json:"pmax"`
}

func (s *BoundsSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "bounds", "pmin", "pmax"); err != nil {
		return err
	}
	type plain BoundsSpec
	return json.Unmarshal(data, (*plain)(s))
}

type MediumBoundarySpec struct {
	Inside   string `json:"inside"`
	Outside  string `json:"outside"`
	Priority *int   `json:"priority,omitempty"`
	Thin     bool   `json:"thin,omitempty"`
}

func (s *MediumBoundarySpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "medium_boundary", "inside", "outside", "priority", "thin"); err != nil {
		return err
	}
	type plain MediumBoundarySpec
	return json.Unmarshal(data, (*plain)(s))
}

// ObjectSpec is the typed Engine object protocol. Raw JSON remains only at
// genuinely polymorphic leaves: expressions, coefficient encodings and
// parameterized field/surface/curve definitions.
type ObjectSpec struct {
	ID             string              `json:"id,omitempty"`
	MaterialID     string              `json:"material_id"`
	Shape          ShapeKind           `json:"shape"`
	MediumBoundary *MediumBoundarySpec `json:"medium_boundary,omitempty"`
	Bounds         *BoundsSpec         `json:"bounds,omitempty"`

	PMin      []float64 `json:"pmin,omitempty"`
	PMax      []float64 `json:"pmax,omitempty"`
	Center    []float64 `json:"center,omitempty"`
	Normal    []float64 `json:"normal,omitempty"`
	Axis      []float64 `json:"axis,omitempty"`
	P1        []float64 `json:"p1,omitempty"`
	P2        []float64 `json:"p2,omitempty"`
	P3        []float64 `json:"p3,omitempty"`
	B         []float64 `json:"b,omitempty"`
	ZDir      []float64 `json:"z_dir,omitempty"`
	XDir      []float64 `json:"x_dir,omitempty"`
	R         *float64  `json:"r,omitempty"`
	Height    *float64  `json:"height,omitempty"`
	RMajor    *float64  `json:"r_major,omitempty"`
	RMinor    *float64  `json:"r_minor,omitempty"`
	Thickness *float64  `json:"thickness,omitempty"`
	C         *float64  `json:"c,omitempty"`

	A            json.RawMessage `json:"a,omitempty"`
	UpperA       json.RawMessage `json:"A,omitempty"`
	Scale        json.RawMessage `json:"scale,omitempty"`
	Field        json.RawMessage `json:"field,omitempty"`
	Transform    json.RawMessage `json:"transform,omitempty"`
	Basis        json.RawMessage `json:"basis,omitempty"`
	Surface      json.RawMessage `json:"surface,omitempty"`
	Curve        json.RawMessage `json:"curve,omitempty"`
	Coefficients json.RawMessage `json:"coefficients,omitempty"`

	Mode          string    `json:"mode,omitempty"`
	File          string    `json:"file,omitempty"`
	URange        []float64 `json:"u_range,omitempty"`
	VRange        []float64 `json:"v_range,omitempty"`
	TRange        []float64 `json:"t_range,omitempty"`
	ExplicitAxis  *int      `json:"explicit_axis,omitempty"`
	InputDim      *int      `json:"input_dim,omitempty"`
	Degree        *int      `json:"degree,omitempty"`
	Samples       *int      `json:"samples,omitempty"`
	SamplesU      *int      `json:"samples_u,omitempty"`
	SamplesV      *int      `json:"samples_v,omitempty"`
	RefineIter    *int      `json:"refine_iter,omitempty"`
	NewtonMaxIter *int      `json:"newton_max_iter,omitempty"`
	Step          *float64  `json:"step,omitempty"`
	ValueTol      *float64  `json:"value_tol,omitempty"`
	NewtonTol     *float64  `json:"newton_tol,omitempty"`
	DerivativeEps *float64  `json:"derivative_eps,omitempty"`
	BoundsPadding *float64  `json:"bounds_padding,omitempty"`
	ResidualTol   *float64  `json:"residual_tol,omitempty"`
}

func (s *ObjectSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "object",
		"id", "material_id", "shape", "medium_boundary", "bounds",
		"pmin", "pmax", "center", "normal", "axis", "p1", "p2", "p3",
		"b", "z_dir", "x_dir", "r", "height", "r_major", "r_minor",
		"thickness", "c", "a", "A", "scale", "field", "transform",
		"basis", "surface", "curve", "coefficients", "mode", "file",
		"u_range", "v_range", "t_range", "explicit_axis", "input_dim",
		"degree", "samples", "samples_u", "samples_v", "refine_iter",
		"newton_max_iter", "step", "value_tol", "newton_tol",
		"derivative_eps", "bounds_padding", "residual_tol"); err != nil {
		return err
	}
	type plain ObjectSpec
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Shape == "" {
		return fmt.Errorf("object: missing required field %q", "shape")
	}
	if !supportedShapeKinds[decoded.Shape] {
		return fmt.Errorf("object: unsupported shape %q", decoded.Shape)
	}
	if err := rejectObjectVariantFields(data, decoded.Shape); err != nil {
		return err
	}
	if decoded.MaterialID == "" {
		return fmt.Errorf("object: missing required field %q", "material_id")
	}
	*s = ObjectSpec(decoded)
	return nil
}

type SurfaceKind string

const (
	SurfaceWeightedMixture             SurfaceKind = "weighted_mixture"
	SurfaceLambert                     SurfaceKind = "lambert"
	SurfaceSpecularReflection          SurfaceKind = "specular_reflection"
	SurfaceSpecularDielectric          SurfaceKind = "specular_dielectric"
	SurfaceRoughConductor              SurfaceKind = "rough_conductor"
	SurfaceRoughDielectricReflection   SurfaceKind = "rough_dielectric_reflection"
	SurfaceCylindricalGridCutout       SurfaceKind = "cylindrical_grid_cutout"
	SurfaceWireMesh                    SurfaceKind = "wire_mesh"
	SurfaceRoughDielectricTransmission SurfaceKind = "rough_dielectric_transmission"
)

var supportedSurfaceKinds = map[SurfaceKind]bool{
	SurfaceWeightedMixture: true, SurfaceLambert: true,
	SurfaceSpecularReflection: true, SurfaceSpecularDielectric: true,
	SurfaceRoughConductor: true, SurfaceRoughDielectricReflection: true,
	SurfaceCylindricalGridCutout: true, SurfaceWireMesh: true,
	SurfaceRoughDielectricTransmission: true,
}

type WeightedSurfaceSpec struct {
	Weight  float64     `json:"weight"`
	Surface SurfaceSpec `json:"surface"`
}

type IORSpec struct {
	Type string   `json:"type"`
	Eta  *float64 `json:"eta,omitempty"`
	A    *float64 `json:"a,omitempty"`
	B    *float64 `json:"b,omitempty"`
	C    *float64 `json:"c,omitempty"`
}

func (s *IORSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "ior", "type", "eta", "a", "b", "c"); err != nil {
		return err
	}
	type plain IORSpec
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Type != "constant" && decoded.Type != "cauchy" {
		return fmt.Errorf("unsupported ior type %q", decoded.Type)
	}
	*s = IORSpec(decoded)
	return nil
}

// SurfaceSpec is a discriminated surface union. Spectral parameters remain
// RawMessage because their value may be a scalar, RGB vector, sampled curve,
// blackbody, or a typed parameter object.
type SurfaceSpec struct {
	Type            SurfaceKind           `json:"type"`
	Components      []WeightedSurfaceSpec `json:"components,omitempty"`
	Albedo          json.RawMessage       `json:"albedo,omitempty"`
	Reflectance     json.RawMessage       `json:"reflectance,omitempty"`
	Transmittance   json.RawMessage       `json:"transmittance,omitempty"`
	Eta             json.RawMessage       `json:"eta,omitempty"`
	K               json.RawMessage       `json:"k,omitempty"`
	Weight          json.RawMessage       `json:"weight,omitempty"`
	IOR             *IORSpec              `json:"ior,omitempty"`
	EtaInside       *float64              `json:"eta_inside,omitempty"`
	EtaOutside      *float64              `json:"eta_outside,omitempty"`
	Roughness       *float64              `json:"roughness,omitempty"`
	LineSurface     *SurfaceSpec          `json:"line_surface,omitempty"`
	Origin          []float64             `json:"origin,omitempty"`
	Axis            []float64             `json:"axis,omitempty"`
	ReferenceAxis   []float64             `json:"reference_axis,omitempty"`
	LineWidth       *float64              `json:"line_width,omitempty"`
	GapWidth        *float64              `json:"gap_width,omitempty"`
	GapHeight       *float64              `json:"gap_height,omitempty"`
	ReferenceRadius *float64              `json:"reference_radius,omitempty"`
}

func (s *SurfaceSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "surface", "type", "components", "albedo",
		"reflectance", "transmittance", "eta", "k", "weight", "ior",
		"eta_inside", "eta_outside", "roughness", "line_surface", "origin",
		"axis", "reference_axis", "line_width", "gap_width", "gap_height",
		"reference_radius"); err != nil {
		return err
	}
	type plain SurfaceSpec
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if !supportedSurfaceKinds[decoded.Type] {
		return fmt.Errorf("unsupported surface type %q", decoded.Type)
	}
	if err := rejectSurfaceVariantFields(data, decoded.Type); err != nil {
		return err
	}
	*s = SurfaceSpec(decoded)
	return nil
}

type EmissionKind string

const (
	EmissionConstant    EmissionKind = "constant"
	EmissionCellPalette EmissionKind = "cell_palette"
	EmissionUVKlein     EmissionKind = "uv_klein"
)

type EmissionDistributionSpec struct {
	Type             string   `json:"type"`
	Sidedness        string   `json:"sidedness,omitempty"`
	Exponent         *float64 `json:"exponent,omitempty"`
	HalfAngleDegrees *float64 `json:"half_angle_degrees,omitempty"`
}

func (s *EmissionDistributionSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "emission distribution", "type", "sidedness", "exponent", "half_angle_degrees"); err != nil {
		return err
	}
	type plain EmissionDistributionSpec
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Type != "uniform" && decoded.Type != "cosine_power" {
		return fmt.Errorf("unsupported emission distribution type %q", decoded.Type)
	}
	*s = EmissionDistributionSpec(decoded)
	return nil
}

type EmissionSpec struct {
	Type          EmissionKind              `json:"type"`
	Radiance      json.RawMessage           `json:"radiance,omitempty"`
	Color         json.RawMessage           `json:"color,omitempty"`
	Exitance      json.RawMessage           `json:"exitance,omitempty"`
	Distribution  *EmissionDistributionSpec `json:"distribution,omitempty"`
	Palette       [][]float64               `json:"palette,omitempty"`
	GridColor     []float64                 `json:"grid_color,omitempty"`
	Intensity     *float64                  `json:"intensity,omitempty"`
	Saturation    *float64                  `json:"saturation,omitempty"`
	Lightness     *float64                  `json:"lightness,omitempty"`
	VStripes      *float64                  `json:"v_stripes,omitempty"`
	Shading       string                    `json:"shading,omitempty"`
	GridThickness *float64                  `json:"grid_thickness,omitempty"`
}

func (s *EmissionSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "emission", "type", "radiance", "color",
		"exitance", "distribution", "palette", "grid_color", "intensity",
		"saturation", "lightness", "v_stripes", "shading", "grid_thickness"); err != nil {
		return err
	}
	type plain EmissionSpec
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	switch decoded.Type {
	case EmissionConstant, EmissionCellPalette, EmissionUVKlein:
	default:
		return fmt.Errorf("unsupported emission type %q", decoded.Type)
	}
	if err := rejectEmissionVariantFields(data, decoded.Type); err != nil {
		return err
	}
	*s = EmissionSpec(decoded)
	return nil
}

type MaterialSpec struct {
	ID       string        `json:"id"`
	Surface  *SurfaceSpec  `json:"surface,omitempty"`
	Emission *EmissionSpec `json:"emission,omitempty"`
}

func (s *MaterialSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "material", "id", "surface", "emission"); err != nil {
		return err
	}
	type plain MaterialSpec
	var decoded plain
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.ID == "" {
		return fmt.Errorf("material: missing required field %q", "id")
	}
	if decoded.Surface == nil && decoded.Emission == nil {
		return fmt.Errorf("material %q requires surface or emission", decoded.ID)
	}
	*s = MaterialSpec(decoded)
	return nil
}

func rejectSpecFields(data []byte, kind string, allowed ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	known := make(map[string]bool, len(allowed))
	for _, field := range allowed {
		known[field] = true
	}
	for field := range fields {
		if !known[field] {
			return fmt.Errorf("unsupported %s field %q", kind, field)
		}
	}
	return nil
}

func rejectObjectVariantFields(data []byte, kind ShapeKind) error {
	base := []string{"id", "material_id", "shape", "medium_boundary", "bounds"}
	var fields []string
	switch kind {
	case ShapeCuboid, ShapeHypercuboid:
		fields = []string{"pmin", "pmax"}
	case ShapeSphere, ShapeHypersphere:
		fields = []string{"center", "r"}
	case ShapeCircle:
		fields = []string{"center", "normal", "r"}
	case ShapeCylinder, ShapeFiniteCylinder:
		fields = []string{"center", "axis", "r", "height"}
	case ShapeTriangle:
		fields = []string{"p1", "p2", "p3"}
	case ShapePlane:
	case ShapeQuadraticEquation:
		fields = []string{"a", "b", "c"}
	case ShapeCubicEquation, ShapeFourOrderEquation:
		fields = []string{"a", "A"}
	case ShapeImplicitEquation:
		fields = []string{"field", "transform", "basis", "step", "value_tol"}
	case ShapeParametricEquation:
		fields = []string{"surface", "u_range", "v_range", "samples_u", "samples_v", "newton_max_iter", "newton_tol", "derivative_eps", "bounds_padding", "residual_tol"}
	case ShapeParametricCurve:
		fields = []string{"curve", "t_range", "samples", "refine_iter", "derivative_eps", "bounds_padding"}
	case ShapePolynomialSurface:
		fields = []string{"mode", "explicit_axis", "input_dim", "degree", "coefficients", "transform", "center", "scale"}
	case ShapeKleinBottle:
		fields = []string{"center", "r_major", "r_minor", "thickness"}
	case ShapeSTL:
		fields = []string{"file", "center", "z_dir", "x_dir", "scale"}
	}
	return rejectSpecFields(data, fmt.Sprintf("object shape %q", kind), append(base, fields...)...)
}

func rejectSurfaceVariantFields(data []byte, kind SurfaceKind) error {
	fields := []string{"type"}
	switch kind {
	case SurfaceWeightedMixture:
		fields = append(fields, "components")
	case SurfaceLambert:
		fields = append(fields, "albedo")
	case SurfaceSpecularReflection:
		fields = append(fields, "reflectance")
	case SurfaceSpecularDielectric:
		fields = append(fields, "reflectance", "transmittance", "eta_inside", "eta_outside", "ior")
	case SurfaceRoughConductor:
		fields = append(fields, "eta", "k", "roughness", "weight")
	case SurfaceRoughDielectricReflection:
		fields = append(fields, "reflectance", "eta_inside", "eta_outside", "ior", "roughness")
	case SurfaceRoughDielectricTransmission:
		fields = append(fields, "transmittance", "eta_inside", "eta_outside", "ior", "roughness")
	case SurfaceCylindricalGridCutout, SurfaceWireMesh:
		fields = append(fields, "line_surface", "origin", "axis", "reference_axis", "line_width", "gap_width", "gap_height", "reference_radius")
	}
	return rejectSpecFields(data, fmt.Sprintf("surface %q", kind), fields...)
}

func rejectEmissionVariantFields(data []byte, kind EmissionKind) error {
	fields := []string{"type", "distribution"}
	switch kind {
	case EmissionConstant:
		fields = append(fields, "radiance", "color", "exitance")
	case EmissionCellPalette:
		fields = append(fields, "palette", "intensity", "shading", "grid_color", "grid_thickness")
	case EmissionUVKlein:
		fields = append(fields, "saturation", "lightness", "v_stripes", "intensity")
	}
	return rejectSpecFields(data, fmt.Sprintf("emission %q", kind), fields...)
}
