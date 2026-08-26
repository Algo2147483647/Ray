package parser

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
)

// ShapeKind is the discriminator for an Engine object definition.
type ShapeKind string

type ObjectCompiler interface {
	CompileCuboid(*CuboidSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileSphere(*SphereSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileCircle(*CircleSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileCylinder(*FiniteCylinderSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileTriangle(*TriangleSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompilePolynomial(*PolynomialSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileImplicitEquation(*ImplicitEquationSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileParametricEquation(*ParametricEquationSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileParametricCurve(*ParametricCurveSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileKleinBottle(*KleinBottleSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
	CompileSTL(*STLSpec, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
}

type ObjectVariantDescriptor struct {
	Kind    ShapeKind
	NewSpec func() ObjectDefinition
	Compile func(ObjectCompiler, ObjectDefinition, *BoundsSpec, geometry.SceneSpace) ([]shape.Shape, error)
}

var objectVariants = make(map[ShapeKind]ObjectVariantDescriptor)

func registerObjectVariant(descriptor ObjectVariantDescriptor) ObjectVariantDescriptor {
	if descriptor.Kind == "" || descriptor.NewSpec == nil || descriptor.Compile == nil {
		panic("invalid object variant descriptor")
	}
	if _, exists := objectVariants[descriptor.Kind]; exists {
		panic(fmt.Sprintf("duplicate object variant %q", descriptor.Kind))
	}
	objectVariants[descriptor.Kind] = descriptor
	return descriptor
}

var (
	ShapeCuboid = registerObjectVariant(ObjectVariantDescriptor{Kind: "cuboid", NewSpec: func() ObjectDefinition { return &CuboidSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileCuboid(d.(*CuboidSpec), b, s)
	}})
	ShapeSphere = registerObjectVariant(ObjectVariantDescriptor{Kind: "sphere", NewSpec: func() ObjectDefinition { return &SphereSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileSphere(d.(*SphereSpec), b, s)
	}})
	ShapeCircle = registerObjectVariant(ObjectVariantDescriptor{Kind: "circle", NewSpec: func() ObjectDefinition { return &CircleSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileCircle(d.(*CircleSpec), b, s)
	}})
	ShapeCylinder = registerObjectVariant(ObjectVariantDescriptor{Kind: "cylinder", NewSpec: func() ObjectDefinition { return &FiniteCylinderSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileCylinder(d.(*FiniteCylinderSpec), b, s)
	}})
	ShapeTriangle = registerObjectVariant(ObjectVariantDescriptor{Kind: "triangle", NewSpec: func() ObjectDefinition { return &TriangleSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileTriangle(d.(*TriangleSpec), b, s)
	}})
	ShapePolynomial = registerObjectVariant(ObjectVariantDescriptor{Kind: "polynomial", NewSpec: func() ObjectDefinition { return &PolynomialSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompilePolynomial(d.(*PolynomialSpec), b, s)
	}})
	ShapeImplicitEquation = registerObjectVariant(ObjectVariantDescriptor{Kind: "implicit equation", NewSpec: func() ObjectDefinition { return &ImplicitEquationSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileImplicitEquation(d.(*ImplicitEquationSpec), b, s)
	}})
	ShapeParametricEquation = registerObjectVariant(ObjectVariantDescriptor{Kind: "parametric equation", NewSpec: func() ObjectDefinition { return &ParametricEquationSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileParametricEquation(d.(*ParametricEquationSpec), b, s)
	}})
	ShapeParametricCurve = registerObjectVariant(ObjectVariantDescriptor{Kind: "parametric curve", NewSpec: func() ObjectDefinition { return &ParametricCurveSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileParametricCurve(d.(*ParametricCurveSpec), b, s)
	}})
	ShapeKleinBottle = registerObjectVariant(ObjectVariantDescriptor{Kind: "klein_bottle", NewSpec: func() ObjectDefinition { return &KleinBottleSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileKleinBottle(d.(*KleinBottleSpec), b, s)
	}})
	ShapeSTL = registerObjectVariant(ObjectVariantDescriptor{Kind: "stl", NewSpec: func() ObjectDefinition { return &STLSpec{} }, Compile: func(c ObjectCompiler, d ObjectDefinition, b *BoundsSpec, s geometry.SceneSpace) ([]shape.Shape, error) {
		return c.CompileSTL(d.(*STLSpec), b, s)
	}})
)

func CompileObjectSpec(compiler ObjectCompiler, spec ObjectSpec, space geometry.SceneSpace) ([]shape.Shape, error) {
	descriptor, ok := objectVariants[spec.Shape]
	if !ok {
		return nil, fmt.Errorf("unsupported shape %q", spec.Shape)
	}
	return descriptor.Compile(compiler, spec.Definition, spec.Bounds, space)
}

type BoundsSpec struct {
	PMin []float64 `json:"pmin,omitempty"`
	PMax []float64 `json:"pmax,omitempty"`
}

func (s *BoundsSpec) UnmarshalJSON(data []byte) error {
	type plain BoundsSpec
	return utils.DecodeStrictJSON(data, "bounds", (*plain)(s))
}

type MediumBoundarySpec struct {
	Inside   string `json:"inside"`
	Outside  string `json:"outside"`
	Priority *int   `json:"priority,omitempty"`
	Thin     bool   `json:"thin,omitempty"`
}

func (s *MediumBoundarySpec) UnmarshalJSON(data []byte) error {
	type plain MediumBoundarySpec
	return utils.DecodeStrictJSON(data, "medium_boundary", (*plain)(s))
}

// ObjectSpec is a discriminated union. Definition always contains the concrete
// spec selected by the variant descriptor above.
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
type PolynomialTermSpec struct {
	Exponents   []int    `json:"exponents"`
	Coefficient *float64 `json:"coefficient"`
}

func (s *PolynomialTermSpec) UnmarshalJSON(data []byte) error {
	type plain PolynomialTermSpec
	var decoded plain
	if err := utils.DecodeStrictJSON(data, "polynomial term", &decoded); err != nil {
		return err
	}
	if len(decoded.Exponents) != 3 {
		return fmt.Errorf("polynomial term exponents must contain 3 values, got %d", len(decoded.Exponents))
	}
	if decoded.Coefficient == nil {
		return fmt.Errorf("polynomial term missing required field %q", "coefficient")
	}
	for axis, exponent := range decoded.Exponents {
		if exponent < 0 {
			return fmt.Errorf("polynomial term exponent[%d] must be >= 0", axis)
		}
	}
	*s = PolynomialTermSpec(decoded)
	return nil
}

type PolynomialSpec struct {
	Degree    *int                 `json:"degree,omitempty"`
	Terms     []PolynomialTermSpec `json:"terms,omitempty"`
	Transform json.RawMessage      `json:"transform,omitempty"`
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
func (*PolynomialSpec) objectDefinition()         {}
func (*ImplicitEquationSpec) objectDefinition()   {}
func (*ParametricEquationSpec) objectDefinition() {}
func (*ParametricCurveSpec) objectDefinition()    {}
func (*KleinBottleSpec) objectDefinition()        {}
func (*STLSpec) objectDefinition()                {}

func (s *ObjectSpec) UnmarshalJSON(data []byte) error {
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
	descriptor, supported := objectVariants[header.Shape]
	if !supported {
		return fmt.Errorf("object: unsupported shape %q", header.Shape)
	}
	if header.MaterialID == "" {
		return fmt.Errorf("object: missing required field %q", "material_id")
	}
	definition := descriptor.NewSpec()
	if err := utils.RejectUnknownJSONFieldsFor(data, "object", &header, definition); err != nil {
		return err
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
	type plain IORSpec
	var decoded plain
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	var allowed []string
	switch header.Type {
	case "constant":
		allowed = []string{"type", "eta"}
	case "cauchy":
		allowed = []string{"type", "a", "b", "c"}
	default:
		return fmt.Errorf("unsupported ior type %q", header.Type)
	}
	if err := utils.RejectUnknownJSONFields(data, "ior", allowed...); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*s = IORSpec(decoded)
	return nil
}

type SurfaceSpec struct {
	Type       SurfaceKind       `json:"type"`
	Definition SurfaceDefinition `json:"-"`
}

type SurfaceDefinition interface {
	surfaceDefinition()
}
type WeightedMixtureSurfaceSpec struct {
	Components []WeightedSurfaceSpec `json:"components,omitempty"`
}
type LambertSurfaceSpec struct {
	Albedo json.RawMessage `json:"albedo,omitempty"`
}
type SpecularReflectionSurfaceSpec struct {
	Reflectance json.RawMessage `json:"reflectance,omitempty"`
}
type SpecularDielectricSurfaceSpec struct {
	Reflectance   json.RawMessage `json:"reflectance,omitempty"`
	Transmittance json.RawMessage `json:"transmittance,omitempty"`
	IOR           *IORSpec        `json:"ior,omitempty"`
	EtaInside     *float64        `json:"eta_inside,omitempty"`
	EtaOutside    *float64        `json:"eta_outside,omitempty"`
}
type RoughConductorSurfaceSpec struct {
	Eta       json.RawMessage `json:"eta,omitempty"`
	K         json.RawMessage `json:"k,omitempty"`
	Weight    json.RawMessage `json:"weight,omitempty"`
	Roughness *float64        `json:"roughness,omitempty"`
}
type RoughDielectricReflectionSurfaceSpec struct {
	Reflectance json.RawMessage `json:"reflectance,omitempty"`
	IOR         *IORSpec        `json:"ior,omitempty"`
	EtaInside   *float64        `json:"eta_inside,omitempty"`
	EtaOutside  *float64        `json:"eta_outside,omitempty"`
	Roughness   *float64        `json:"roughness,omitempty"`
}
type RoughDielectricTransmissionSurfaceSpec struct {
	Transmittance json.RawMessage `json:"transmittance,omitempty"`
	IOR           *IORSpec        `json:"ior,omitempty"`
	EtaInside     *float64        `json:"eta_inside,omitempty"`
	EtaOutside    *float64        `json:"eta_outside,omitempty"`
	Roughness     *float64        `json:"roughness,omitempty"`
}
type CylindricalGridSurfaceSpec struct {
	LineSurface     *SurfaceSpec `json:"line_surface,omitempty"`
	Origin          []float64    `json:"origin,omitempty"`
	Axis            []float64    `json:"axis,omitempty"`
	ReferenceAxis   []float64    `json:"reference_axis,omitempty"`
	LineWidth       *float64     `json:"line_width,omitempty"`
	GapWidth        *float64     `json:"gap_width,omitempty"`
	GapHeight       *float64     `json:"gap_height,omitempty"`
	ReferenceRadius *float64     `json:"reference_radius,omitempty"`
}

func (*WeightedMixtureSurfaceSpec) surfaceDefinition()             {}
func (*LambertSurfaceSpec) surfaceDefinition()                     {}
func (*SpecularReflectionSurfaceSpec) surfaceDefinition()          {}
func (*SpecularDielectricSurfaceSpec) surfaceDefinition()          {}
func (*RoughConductorSurfaceSpec) surfaceDefinition()              {}
func (*RoughDielectricReflectionSurfaceSpec) surfaceDefinition()   {}
func (*RoughDielectricTransmissionSurfaceSpec) surfaceDefinition() {}
func (*CylindricalGridSurfaceSpec) surfaceDefinition()             {}

type SurfaceCompiler interface {
	CompileWeightedMixture(*WeightedMixtureSurfaceSpec) (bxdf.Scattering, error)
	CompileLambert(*LambertSurfaceSpec) (bxdf.Scattering, error)
	CompileSpecularReflection(*SpecularReflectionSurfaceSpec) (bxdf.Scattering, error)
	CompileSpecularDielectric(*SpecularDielectricSurfaceSpec) (bxdf.Scattering, error)
	CompileRoughConductor(*RoughConductorSurfaceSpec) (bxdf.Scattering, error)
	CompileRoughDielectricReflection(*RoughDielectricReflectionSurfaceSpec) (bxdf.Scattering, error)
	CompileRoughDielectricTransmission(*RoughDielectricTransmissionSurfaceSpec) (bxdf.Scattering, error)
	CompileCylindricalGrid(*CylindricalGridSurfaceSpec) (bxdf.Scattering, error)
}

type SurfaceVariantDescriptor struct {
	Kind    SurfaceKind
	NewSpec func() SurfaceDefinition
	Compile func(SurfaceCompiler, SurfaceDefinition) (bxdf.Scattering, error)
}

var surfaceVariants = map[SurfaceKind]SurfaceVariantDescriptor{}

func registerSurfaceVariant(descriptor SurfaceVariantDescriptor) SurfaceVariantDescriptor {
	if descriptor.Kind == "" || descriptor.NewSpec == nil || descriptor.Compile == nil {
		panic("invalid surface variant descriptor")
	}
	if _, exists := surfaceVariants[descriptor.Kind]; exists {
		panic(fmt.Sprintf("duplicate surface variant %q", descriptor.Kind))
	}
	surfaceVariants[descriptor.Kind] = descriptor
	return descriptor
}

var (
	SurfaceWeightedMixture = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "weighted_mixture", NewSpec: func() SurfaceDefinition { return &WeightedMixtureSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileWeightedMixture(d.(*WeightedMixtureSurfaceSpec))
	}})
	SurfaceLambert = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "lambert", NewSpec: func() SurfaceDefinition { return &LambertSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileLambert(d.(*LambertSurfaceSpec))
	}})
	SurfaceSpecularReflection = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "specular_reflection", NewSpec: func() SurfaceDefinition { return &SpecularReflectionSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileSpecularReflection(d.(*SpecularReflectionSurfaceSpec))
	}})
	SurfaceSpecularDielectric = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "specular_dielectric", NewSpec: func() SurfaceDefinition { return &SpecularDielectricSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileSpecularDielectric(d.(*SpecularDielectricSurfaceSpec))
	}})
	SurfaceRoughConductor = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "rough_conductor", NewSpec: func() SurfaceDefinition { return &RoughConductorSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileRoughConductor(d.(*RoughConductorSurfaceSpec))
	}})
	SurfaceRoughDielectricReflection = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "rough_dielectric_reflection", NewSpec: func() SurfaceDefinition { return &RoughDielectricReflectionSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileRoughDielectricReflection(d.(*RoughDielectricReflectionSurfaceSpec))
	}})
	SurfaceRoughDielectricTransmission = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "rough_dielectric_transmission", NewSpec: func() SurfaceDefinition { return &RoughDielectricTransmissionSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileRoughDielectricTransmission(d.(*RoughDielectricTransmissionSurfaceSpec))
	}})
	SurfaceCylindricalGridCutout = registerSurfaceVariant(SurfaceVariantDescriptor{Kind: "cylindrical_grid_cutout", NewSpec: func() SurfaceDefinition { return &CylindricalGridSurfaceSpec{} }, Compile: func(c SurfaceCompiler, d SurfaceDefinition) (bxdf.Scattering, error) {
		return c.CompileCylindricalGrid(d.(*CylindricalGridSurfaceSpec))
	}})
)

func CompileSurfaceSpec(compiler SurfaceCompiler, spec *SurfaceSpec) (bxdf.Scattering, error) {
	if spec == nil {
		return nil, fmt.Errorf("surface spec is nil")
	}
	descriptor, ok := surfaceVariants[spec.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported surface type %q", spec.Type)
	}
	return descriptor.Compile(compiler, spec.Definition)
}

func (s *SurfaceSpec) UnmarshalJSON(data []byte) error {
	var header struct {
		Type SurfaceKind `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	descriptor, ok := surfaceVariants[header.Type]
	if !ok {
		return fmt.Errorf("unsupported surface type %q", header.Type)
	}
	definition := descriptor.NewSpec()
	if err := utils.RejectUnknownJSONFieldsFor(data, "surface", &header, definition); err != nil {
		return err
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

type EmissionDistributionSpec struct {
	Type             string   `json:"type"`
	Sidedness        string   `json:"sidedness,omitempty"`
	Exponent         *float64 `json:"exponent,omitempty"`
	HalfAngleDegrees *float64 `json:"half_angle_degrees,omitempty"`
}

func (s *EmissionDistributionSpec) UnmarshalJSON(data []byte) error {
	type plain EmissionDistributionSpec
	var decoded plain
	if err := utils.DecodeStrictJSON(data, "emission distribution", &decoded); err != nil {
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

type EmissionDefinition interface {
	emissionDefinition()
}
type ConstantEmissionSpec struct {
	Radiance json.RawMessage `json:"radiance,omitempty"`
	Color    json.RawMessage `json:"color,omitempty"`
	Exitance json.RawMessage `json:"exitance,omitempty"`
}
type NormalPaletteEmissionSpec struct {
	Palette   [][]float64 `json:"palette,omitempty"`
	Intensity *float64    `json:"intensity,omitempty"`
}
type UVHSLEmissionSpec struct {
	Saturation *float64 `json:"saturation,omitempty"`
	Lightness  *float64 `json:"lightness,omitempty"`
	VStripes   *float64 `json:"v_stripes,omitempty"`
	Intensity  *float64 `json:"intensity,omitempty"`
}

func (*ConstantEmissionSpec) emissionDefinition()      {}
func (*NormalPaletteEmissionSpec) emissionDefinition() {}
func (*UVHSLEmissionSpec) emissionDefinition()         {}

type EmissionCompiler interface {
	CompileConstantEmission(*ConstantEmissionSpec, *EmissionDistributionSpec) (emission.Emitter, error)
	CompileNormalPaletteEmission(*NormalPaletteEmissionSpec, *EmissionDistributionSpec) (emission.Emitter, error)
	CompileUVHSLEmission(*UVHSLEmissionSpec, *EmissionDistributionSpec) (emission.Emitter, error)
}

type EmissionVariantDescriptor struct {
	Kind    EmissionKind
	NewSpec func() EmissionDefinition
	Compile func(EmissionCompiler, EmissionDefinition, *EmissionDistributionSpec) (emission.Emitter, error)
}

var emissionVariants = map[EmissionKind]EmissionVariantDescriptor{}

func registerEmissionVariant(descriptor EmissionVariantDescriptor) EmissionVariantDescriptor {
	if descriptor.Kind == "" || descriptor.NewSpec == nil || descriptor.Compile == nil {
		panic("invalid emission variant descriptor")
	}
	if _, exists := emissionVariants[descriptor.Kind]; exists {
		panic(fmt.Sprintf("duplicate emission variant %q", descriptor.Kind))
	}
	emissionVariants[descriptor.Kind] = descriptor
	return descriptor
}

var (
	EmissionConstant = registerEmissionVariant(EmissionVariantDescriptor{Kind: "constant", NewSpec: func() EmissionDefinition { return &ConstantEmissionSpec{} }, Compile: func(c EmissionCompiler, d EmissionDefinition, distribution *EmissionDistributionSpec) (emission.Emitter, error) {
		return c.CompileConstantEmission(d.(*ConstantEmissionSpec), distribution)
	}})
	EmissionNormalPalette = registerEmissionVariant(EmissionVariantDescriptor{Kind: "normal_palette", NewSpec: func() EmissionDefinition { return &NormalPaletteEmissionSpec{} }, Compile: func(c EmissionCompiler, d EmissionDefinition, distribution *EmissionDistributionSpec) (emission.Emitter, error) {
		return c.CompileNormalPaletteEmission(d.(*NormalPaletteEmissionSpec), distribution)
	}})
	EmissionUVHSL = registerEmissionVariant(EmissionVariantDescriptor{Kind: "uv_hsl", NewSpec: func() EmissionDefinition { return &UVHSLEmissionSpec{} }, Compile: func(c EmissionCompiler, d EmissionDefinition, distribution *EmissionDistributionSpec) (emission.Emitter, error) {
		return c.CompileUVHSLEmission(d.(*UVHSLEmissionSpec), distribution)
	}})
)

func CompileEmissionSpec(compiler EmissionCompiler, spec *EmissionSpec) (emission.Emitter, error) {
	if spec == nil {
		return nil, fmt.Errorf("emission spec is nil")
	}
	descriptor, ok := emissionVariants[spec.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported emission type %q", spec.Type)
	}
	return descriptor.Compile(compiler, spec.Definition, spec.Distribution)
}

func (s *EmissionSpec) UnmarshalJSON(data []byte) error {
	var header struct {
		Type         EmissionKind              `json:"type"`
		Distribution *EmissionDistributionSpec `json:"distribution"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return err
	}
	descriptor, ok := emissionVariants[header.Type]
	if !ok {
		return fmt.Errorf("unsupported emission type %q", header.Type)
	}
	definition := descriptor.NewSpec()
	if err := utils.RejectUnknownJSONFieldsFor(data, "emission", &header, definition); err != nil {
		return err
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
	type plain MaterialSpec
	var decoded plain
	if err := utils.DecodeStrictJSON(data, "material", &decoded); err != nil {
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
