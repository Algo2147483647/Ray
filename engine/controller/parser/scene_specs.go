package parser

import (
	"bytes"
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
	PMin []float64 `json:"pmin,omitempty"`
	PMax []float64 `json:"pmax,omitempty"`
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

// ObjectSpec is a discriminated union. Definition always contains one of the
// concrete shape specs below; builders must switch on that concrete type rather
// than converting the protocol back to a string-keyed map.
type ObjectSpec struct {
	ID             string              `json:"id,omitempty"`
	MaterialID     string              `json:"material_id"`
	Shape          ShapeKind           `json:"shape"`
	MediumBoundary *MediumBoundarySpec `json:"medium_boundary,omitempty"`
	Bounds         *BoundsSpec         `json:"bounds,omitempty"`
	Definition     ObjectDefinition    `json:"-"`
}

type ObjectDefinition interface{ objectDefinition() }

type CuboidSpec struct {
	PMin []float64 `json:"pmin,omitempty"`
	PMax []float64 `json:"pmax,omitempty"`
}
type SphereSpec struct {
	Center []float64 `json:"center,omitempty"`
	R      *float64  `json:"r,omitempty"`
}
type CircleSpec struct {
	Center   []float64 `json:"center,omitempty"`
	Position []float64 `json:"position,omitempty"`
	Normal   []float64 `json:"normal,omitempty"`
	R        *float64  `json:"r,omitempty"`
}
type FiniteCylinderSpec struct {
	Center   []float64 `json:"center,omitempty"`
	Position []float64 `json:"position,omitempty"`
	Axis     []float64 `json:"axis,omitempty"`
	R        *float64  `json:"r,omitempty"`
	Height   *float64  `json:"height,omitempty"`
}
type TriangleSpec struct {
	P1 []float64 `json:"p1,omitempty"`
	P2 []float64 `json:"p2,omitempty"`
	P3 []float64 `json:"p3,omitempty"`
}
type PlaneSpec struct{}
type QuadraticEquationSpec struct {
	A []float64 `json:"a,omitempty"`
	B []float64 `json:"b,omitempty"`
	C *float64  `json:"c,omitempty"`
}
type PolynomialEquationSpec struct {
	A      json.RawMessage `json:"a,omitempty"`
	UpperA json.RawMessage `json:"A,omitempty"`
	Center []float64       `json:"center,omitempty"`
	Scale  json.RawMessage `json:"scale,omitempty"`
	Basis  json.RawMessage `json:"basis,omitempty"`
}
type ImplicitEquationSpec struct {
	Field     json.RawMessage `json:"field,omitempty"`
	Transform json.RawMessage `json:"transform,omitempty"`
	Basis     json.RawMessage `json:"basis,omitempty"`
	Step      *float64        `json:"step,omitempty"`
	ValueTol  *float64        `json:"value_tol,omitempty"`
	Center    []float64       `json:"center,omitempty"`
	Scale     json.RawMessage `json:"scale,omitempty"`
}
type ParametricEquationSpec struct {
	Surface       json.RawMessage `json:"surface,omitempty"`
	URange        []float64       `json:"u_range,omitempty"`
	VRange        []float64       `json:"v_range,omitempty"`
	SamplesU      *int            `json:"samples_u,omitempty"`
	SamplesV      *int            `json:"samples_v,omitempty"`
	NewtonMaxIter *int            `json:"newton_max_iter,omitempty"`
	NewtonTol     *float64        `json:"newton_tol,omitempty"`
	DerivativeEps *float64        `json:"derivative_eps,omitempty"`
	BoundsPadding *float64        `json:"bounds_padding,omitempty"`
	ResidualTol   *float64        `json:"residual_tol,omitempty"`
	Center        []float64       `json:"center,omitempty"`
	Scale         json.RawMessage `json:"scale,omitempty"`
}
type ParametricCurveSpec struct {
	Curve         json.RawMessage `json:"curve,omitempty"`
	TRange        []float64       `json:"t_range,omitempty"`
	Samples       *int            `json:"samples,omitempty"`
	RefineIter    *int            `json:"refine_iter,omitempty"`
	DerivativeEps *float64        `json:"derivative_eps,omitempty"`
	BoundsPadding *float64        `json:"bounds_padding,omitempty"`
	Center        []float64       `json:"center,omitempty"`
	Scale         json.RawMessage `json:"scale,omitempty"`
}
type PolynomialSurfaceSpec struct {
	Mode         string          `json:"mode,omitempty"`
	ExplicitAxis *int            `json:"explicit_axis,omitempty"`
	InputDim     *int            `json:"input_dim,omitempty"`
	Degree       *int            `json:"degree,omitempty"`
	Coefficients json.RawMessage `json:"coefficients,omitempty"`
	Transform    json.RawMessage `json:"transform,omitempty"`
	Scale        json.RawMessage `json:"scale,omitempty"`
	Center       []float64       `json:"center,omitempty"`
}
type KleinBottleSpec struct {
	Center    []float64 `json:"center,omitempty"`
	RMajor    *float64  `json:"r_major,omitempty"`
	RMinor    *float64  `json:"r_minor,omitempty"`
	Thickness *float64  `json:"thickness,omitempty"`
}
type STLSpec struct {
	File   string    `json:"file,omitempty"`
	Center []float64 `json:"center,omitempty"`
	ZDir   []float64 `json:"z_dir,omitempty"`
	XDir   []float64 `json:"x_dir,omitempty"`
	Scale  []float64 `json:"scale,omitempty"`
}

func (*CuboidSpec) objectDefinition()             {}
func (*SphereSpec) objectDefinition()             {}
func (*CircleSpec) objectDefinition()             {}
func (*FiniteCylinderSpec) objectDefinition()     {}
func (*TriangleSpec) objectDefinition()           {}
func (*PlaneSpec) objectDefinition()              {}
func (*QuadraticEquationSpec) objectDefinition()  {}
func (*PolynomialEquationSpec) objectDefinition() {}
func (*ImplicitEquationSpec) objectDefinition()   {}
func (*ParametricEquationSpec) objectDefinition() {}
func (*ParametricCurveSpec) objectDefinition()    {}
func (*PolynomialSurfaceSpec) objectDefinition()  {}
func (*KleinBottleSpec) objectDefinition()        {}
func (*STLSpec) objectDefinition()                {}

func (s *ObjectSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "object",
		"id", "material_id", "shape", "medium_boundary", "bounds",
		"pmin", "pmax", "center", "position", "normal", "axis", "p1", "p2", "p3",
		"b", "z_dir", "x_dir", "r", "height", "r_major", "r_minor",
		"thickness", "c", "a", "A", "scale", "field", "transform",
		"basis", "surface", "curve", "coefficients", "mode", "file",
		"u_range", "v_range", "t_range", "explicit_axis", "input_dim",
		"degree", "samples", "samples_u", "samples_v", "refine_iter",
		"newton_max_iter", "step", "value_tol", "newton_tol",
		"derivative_eps", "bounds_padding", "residual_tol"); err != nil {
		return err
	}
	var header struct {
		ID             string              `json:"id"`
		MaterialID     string              `json:"material_id"`
		Shape          ShapeKind           `json:"shape"`
		MediumBoundary *MediumBoundarySpec `json:"medium_boundary"`
		Bounds         *BoundsSpec         `json:"bounds"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	if header.Shape == "" {
		return fmt.Errorf("object: missing required field %q", "shape")
	}
	if !supportedShapeKinds[header.Shape] {
		return fmt.Errorf("object: unsupported shape %q", header.Shape)
	}
	if err := rejectObjectVariantFields(data, header.Shape); err != nil {
		return err
	}
	if header.MaterialID == "" {
		return fmt.Errorf("object: missing required field %q", "material_id")
	}
	var definition ObjectDefinition
	switch header.Shape {
	case ShapeCuboid, ShapeHypercuboid:
		definition = &CuboidSpec{}
	case ShapeSphere, ShapeHypersphere:
		definition = &SphereSpec{}
	case ShapeCircle:
		definition = &CircleSpec{}
	case ShapeCylinder, ShapeFiniteCylinder:
		definition = &FiniteCylinderSpec{}
	case ShapeTriangle:
		definition = &TriangleSpec{}
	case ShapePlane:
		definition = &PlaneSpec{}
	case ShapeQuadraticEquation:
		definition = &QuadraticEquationSpec{}
	case ShapeCubicEquation, ShapeFourOrderEquation:
		definition = &PolynomialEquationSpec{}
	case ShapeImplicitEquation:
		definition = &ImplicitEquationSpec{}
	case ShapeParametricEquation:
		definition = &ParametricEquationSpec{}
	case ShapeParametricCurve:
		definition = &ParametricCurveSpec{}
	case ShapePolynomialSurface:
		definition = &PolynomialSurfaceSpec{}
	case ShapeKleinBottle:
		definition = &KleinBottleSpec{}
	case ShapeSTL:
		definition = &STLSpec{}
	}
	if err := json.Unmarshal(data, definition); err != nil {
		return err
	}
	*s = ObjectSpec{ID: header.ID, MaterialID: header.MaterialID, Shape: header.Shape,
		MediumBoundary: header.MediumBoundary, Bounds: header.Bounds, Definition: definition}
	return nil
}

func (s ObjectSpec) MarshalJSON() ([]byte, error) {
	header := struct {
		ID             string              `json:"id,omitempty"`
		MaterialID     string              `json:"material_id"`
		Shape          ShapeKind           `json:"shape"`
		MediumBoundary *MediumBoundarySpec `json:"medium_boundary,omitempty"`
		Bounds         *BoundsSpec         `json:"bounds,omitempty"`
	}{s.ID, s.MaterialID, s.Shape, s.MediumBoundary, s.Bounds}
	return marshalDiscriminated(header, s.Definition)
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

type SurfaceSpec struct {
	Type       SurfaceKind       `json:"type"`
	Definition SurfaceDefinition `json:"-"`
}

type SurfaceDefinition interface{ surfaceDefinition() }
type WeightedMixtureSurfaceSpec struct {
	Components []WeightedSurfaceSpec `json:"components"`
}
type LambertSurfaceSpec struct {
	Albedo json.RawMessage `json:"albedo"`
}
type SpecularReflectionSurfaceSpec struct {
	Reflectance json.RawMessage `json:"reflectance"`
}
type SpecularDielectricSurfaceSpec struct {
	Reflectance   json.RawMessage `json:"reflectance"`
	Transmittance json.RawMessage `json:"transmittance"`
	IOR           *IORSpec        `json:"ior"`
	EtaInside     *float64        `json:"eta_inside"`
	EtaOutside    *float64        `json:"eta_outside"`
}
type RoughConductorSurfaceSpec struct {
	Eta       json.RawMessage `json:"eta"`
	K         json.RawMessage `json:"k"`
	Weight    json.RawMessage `json:"weight"`
	Roughness *float64        `json:"roughness"`
}
type RoughDielectricReflectionSurfaceSpec struct {
	Reflectance json.RawMessage `json:"reflectance"`
	IOR         *IORSpec        `json:"ior"`
	EtaInside   *float64        `json:"eta_inside"`
	EtaOutside  *float64        `json:"eta_outside"`
	Roughness   *float64        `json:"roughness"`
}
type RoughDielectricTransmissionSurfaceSpec struct {
	Transmittance json.RawMessage `json:"transmittance"`
	IOR           *IORSpec        `json:"ior"`
	EtaInside     *float64        `json:"eta_inside"`
	EtaOutside    *float64        `json:"eta_outside"`
	Roughness     *float64        `json:"roughness"`
}
type CylindricalGridSurfaceSpec struct {
	LineSurface     *SurfaceSpec `json:"line_surface"`
	Origin          []float64    `json:"origin"`
	Axis            []float64    `json:"axis"`
	ReferenceAxis   []float64    `json:"reference_axis"`
	LineWidth       *float64     `json:"line_width"`
	GapWidth        *float64     `json:"gap_width"`
	GapHeight       *float64     `json:"gap_height"`
	ReferenceRadius *float64     `json:"reference_radius"`
}

func (*WeightedMixtureSurfaceSpec) surfaceDefinition()             {}
func (*LambertSurfaceSpec) surfaceDefinition()                     {}
func (*SpecularReflectionSurfaceSpec) surfaceDefinition()          {}
func (*SpecularDielectricSurfaceSpec) surfaceDefinition()          {}
func (*RoughConductorSurfaceSpec) surfaceDefinition()              {}
func (*RoughDielectricReflectionSurfaceSpec) surfaceDefinition()   {}
func (*RoughDielectricTransmissionSurfaceSpec) surfaceDefinition() {}
func (*CylindricalGridSurfaceSpec) surfaceDefinition()             {}

func (s *SurfaceSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "surface", "type", "components", "albedo",
		"reflectance", "transmittance", "eta", "k", "weight", "ior",
		"eta_inside", "eta_outside", "roughness", "line_surface", "origin",
		"axis", "reference_axis", "line_width", "gap_width", "gap_height",
		"reference_radius"); err != nil {
		return err
	}
	var header struct {
		Type SurfaceKind `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	if !supportedSurfaceKinds[header.Type] {
		return fmt.Errorf("unsupported surface type %q", header.Type)
	}
	if err := rejectSurfaceVariantFields(data, header.Type); err != nil {
		return err
	}
	var definition SurfaceDefinition
	switch header.Type {
	case SurfaceWeightedMixture:
		definition = &WeightedMixtureSurfaceSpec{}
	case SurfaceLambert:
		definition = &LambertSurfaceSpec{}
	case SurfaceSpecularReflection:
		definition = &SpecularReflectionSurfaceSpec{}
	case SurfaceSpecularDielectric:
		definition = &SpecularDielectricSurfaceSpec{}
	case SurfaceRoughConductor:
		definition = &RoughConductorSurfaceSpec{}
	case SurfaceRoughDielectricReflection:
		definition = &RoughDielectricReflectionSurfaceSpec{}
	case SurfaceRoughDielectricTransmission:
		definition = &RoughDielectricTransmissionSurfaceSpec{}
	case SurfaceCylindricalGridCutout, SurfaceWireMesh:
		definition = &CylindricalGridSurfaceSpec{}
	}
	if err := json.Unmarshal(data, definition); err != nil {
		return err
	}
	*s = SurfaceSpec{Type: header.Type, Definition: definition}
	return nil
}

func (s SurfaceSpec) MarshalJSON() ([]byte, error) {
	return marshalDiscriminated(struct {
		Type SurfaceKind `json:"type"`
	}{s.Type}, s.Definition)
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
	Type         EmissionKind              `json:"type"`
	Distribution *EmissionDistributionSpec `json:"distribution,omitempty"`
	Definition   EmissionDefinition        `json:"-"`
}

type EmissionDefinition interface{ emissionDefinition() }
type ConstantEmissionSpec struct {
	Radiance json.RawMessage `json:"radiance"`
	Color    json.RawMessage `json:"color"`
	Exitance json.RawMessage `json:"exitance"`
}
type CellPaletteEmissionSpec struct {
	Palette       [][]float64 `json:"palette"`
	GridColor     []float64   `json:"grid_color"`
	Intensity     *float64    `json:"intensity"`
	Shading       string      `json:"shading"`
	GridThickness *float64    `json:"grid_thickness"`
}
type UVKleinEmissionSpec struct {
	Saturation *float64 `json:"saturation"`
	Lightness  *float64 `json:"lightness"`
	VStripes   *float64 `json:"v_stripes"`
	Intensity  *float64 `json:"intensity"`
}

func (*ConstantEmissionSpec) emissionDefinition()    {}
func (*CellPaletteEmissionSpec) emissionDefinition() {}
func (*UVKleinEmissionSpec) emissionDefinition()     {}

func (s *EmissionSpec) UnmarshalJSON(data []byte) error {
	if err := rejectSpecFields(data, "emission", "type", "radiance", "color",
		"exitance", "distribution", "palette", "grid_color", "intensity",
		"saturation", "lightness", "v_stripes", "shading", "grid_thickness"); err != nil {
		return err
	}
	var header struct {
		Type         EmissionKind              `json:"type"`
		Distribution *EmissionDistributionSpec `json:"distribution"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	var definition EmissionDefinition
	switch header.Type {
	case EmissionConstant, EmissionCellPalette, EmissionUVKlein:
	default:
		return fmt.Errorf("unsupported emission type %q", header.Type)
	}
	if err := rejectEmissionVariantFields(data, header.Type); err != nil {
		return err
	}
	switch header.Type {
	case EmissionConstant:
		definition = &ConstantEmissionSpec{}
	case EmissionCellPalette:
		definition = &CellPaletteEmissionSpec{}
	case EmissionUVKlein:
		definition = &UVKleinEmissionSpec{}
	}
	if err := json.Unmarshal(data, definition); err != nil {
		return err
	}
	*s = EmissionSpec{Type: header.Type, Distribution: header.Distribution, Definition: definition}
	return nil
}

func (s EmissionSpec) MarshalJSON() ([]byte, error) {
	header := struct {
		Type         EmissionKind              `json:"type"`
		Distribution *EmissionDistributionSpec `json:"distribution,omitempty"`
	}{s.Type, s.Distribution}
	return marshalDiscriminated(header, s.Definition)
}

func marshalDiscriminated(header, definition interface{}) ([]byte, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return headerJSON, nil
	}
	definitionJSON, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}
	headerBody := bytes.TrimSpace(headerJSON)[1 : len(bytes.TrimSpace(headerJSON))-1]
	definitionBody := bytes.TrimSpace(definitionJSON)[1 : len(bytes.TrimSpace(definitionJSON))-1]
	result := make([]byte, 0, len(headerBody)+len(definitionBody)+3)
	result = append(result, '{')
	result = append(result, headerBody...)
	if len(headerBody) > 0 && len(definitionBody) > 0 {
		result = append(result, ',')
	}
	result = append(result, definitionBody...)
	result = append(result, '}')
	return result, nil
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
		fields = []string{"center", "position", "normal", "r"}
	case ShapeCylinder, ShapeFiniteCylinder:
		fields = []string{"center", "position", "axis", "r", "height"}
	case ShapeTriangle:
		fields = []string{"p1", "p2", "p3"}
	case ShapePlane:
	case ShapeQuadraticEquation:
		fields = []string{"a", "b", "c"}
	case ShapeCubicEquation, ShapeFourOrderEquation:
		fields = []string{"a", "A", "center", "scale", "basis"}
	case ShapeImplicitEquation:
		fields = []string{"field", "transform", "basis", "center", "scale", "step", "value_tol"}
	case ShapeParametricEquation:
		fields = []string{"surface", "center", "scale", "u_range", "v_range", "samples_u", "samples_v", "newton_max_iter", "newton_tol", "derivative_eps", "bounds_padding", "residual_tol"}
	case ShapeParametricCurve:
		fields = []string{"curve", "center", "scale", "t_range", "samples", "refine_iter", "derivative_eps", "bounds_padding"}
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
