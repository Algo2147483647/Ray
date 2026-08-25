package factory

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
	"github.com/Algo2147483647/ray/engine/utils"
)

func ParseMaterials(script *parser.Script) (map[string]*material.Material, error) {
	if script == nil {
		return nil, errors.New("script is nil")
	}
	materials := make(map[string]*material.Material, len(script.Materials))
	var parseErrors []error
	for index, spec := range script.Materials {
		context := fmt.Sprintf("material[%d] id=%q", index, spec.ID)
		if _, exists := materials[spec.ID]; exists {
			parseErrors = append(parseErrors, fmt.Errorf("%s: duplicate material id", context))
			continue
		}
		result := &material.Material{Metadata: material.MaterialMetadata{Name: spec.ID, SpectrumMode: optics.SpectrumModeRGB}}
		if spec.Surface != nil {
			surface, err := parseSurface(spec.Surface)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("%s surface: %w", context, err))
				continue
			}
			result.Surface = surface
		}
		if spec.Emission != nil {
			emitter, err := parseEmission(spec.Emission)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("%s emission: %w", context, err))
				continue
			}
			result.Emission = emitter
		}
		if !result.HasSurface() && !result.HasEmission() {
			parseErrors = append(parseErrors, fmt.Errorf("%s: material requires surface or emission", context))
			continue
		}
		materials[spec.ID] = result
	}
	if len(parseErrors) > 0 {
		return nil, errors.Join(parseErrors...)
	}
	return materials, nil
}

func parseSurface(spec *parser.SurfaceSpec) (bxdf.Scattering, error) {
	switch definition := spec.Definition.(type) {
	case *parser.WeightedMixtureSurfaceSpec:
		if len(definition.Components) == 0 {
			return nil, fmt.Errorf("field %q must not be empty", "components")
		}
		components := make([]bsdf.WeightedScattering, 0, len(definition.Components))
		for index, component := range definition.Components {
			if component.Weight <= 0 || math.IsNaN(component.Weight) || math.IsInf(component.Weight, 0) {
				return nil, fmt.Errorf("components[%d] weight must be finite and > 0", index)
			}
			surface, err := parseSurface(&component.Surface)
			if err != nil {
				return nil, fmt.Errorf("components[%d] surface: %w", index, err)
			}
			components = append(components, bsdf.WeightedScattering{Weight: component.Weight, Scattering: surface})
		}
		return bsdf.NewWeightedMixture(components...), nil
	case *parser.LambertSurfaceSpec:
		albedo, err := requiredSpectralParameterRaw(definition.Albedo, "albedo")
		if err != nil {
			return nil, err
		}
		return bxdf.NewLambertParameter(albedo), nil
	case *parser.SpecularReflectionSurfaceSpec:
		reflectance, err := optionalSpectralParameterRaw(definition.Reflectance, "reflectance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		return bxdf.NewSpecularReflectionParameter(reflectance), nil
	case *parser.SpecularDielectricSurfaceSpec:
		reflectance, err := optionalSpectralParameterRaw(definition.Reflectance, "reflectance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		transmittance, err := optionalSpectralParameterRaw(definition.Transmittance, "transmittance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		outside, err := parseEtaOutside(definition.EtaOutside)
		if err != nil {
			return nil, err
		}
		inside, err := parseIORModel(definition.IOR, definition.EtaInside)
		if err != nil {
			return nil, err
		}
		return bxdf.NewSpecularDielectricParameter(reflectance, transmittance, outside, inside), nil
	case *parser.RoughConductorSurfaceSpec:
		eta, err := requiredSpectralParameterRaw(definition.Eta, "eta")
		if err != nil {
			return nil, err
		}
		k, err := requiredSpectralParameterRaw(definition.K, "k")
		if err != nil {
			return nil, err
		}
		rough, err := parseRoughness(definition.Roughness)
		if err != nil {
			return nil, err
		}
		weight, err := optionalSpectralParameterRaw(definition.Weight, "weight", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		result := bxdf.NewRoughConductorParameter(eta, k, rough*rough)
		result.Weight = weight
		return result, nil
	case *parser.RoughDielectricReflectionSurfaceSpec:
		reflectance, err := optionalSpectralParameterRaw(definition.Reflectance, "reflectance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		outside, err := parseEtaOutside(definition.EtaOutside)
		if err != nil {
			return nil, err
		}
		inside, err := parseIORModel(definition.IOR, definition.EtaInside)
		if err != nil {
			return nil, err
		}
		rough, err := parseRoughness(definition.Roughness)
		if err != nil {
			return nil, err
		}
		return bxdf.NewRoughDielectricReflectionParameter(reflectance, outside, inside, rough*rough), nil
	case *parser.RoughDielectricTransmissionSurfaceSpec:
		transmittance, err := optionalSpectralParameterRaw(definition.Transmittance, "transmittance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		outside, err := parseEtaOutside(definition.EtaOutside)
		if err != nil {
			return nil, err
		}
		inside, err := parseIORModel(definition.IOR, definition.EtaInside)
		if err != nil {
			return nil, err
		}
		rough, err := parseRoughness(definition.Roughness)
		if err != nil {
			return nil, err
		}
		return bxdf.NewRoughDielectricTransmissionParameter(transmittance, outside, inside, rough*rough), nil
	case *parser.CylindricalGridSurfaceSpec:
		return parseCylindricalGridCutoutSurface(definition)
	default:
		return nil, fmt.Errorf("unsupported surface type %q", spec.Type)
	}
}

func parseRoughness(value *float64) (float64, error) {
	result := 0.25
	if value != nil {
		result = *value
	}
	if result < 0 || result > 1 || math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, fmt.Errorf("roughness must be in [0, 1]")
	}
	return result, nil
}
func parseEtaOutside(value *float64) (float64, error) {
	result := 1.0
	if value != nil {
		result = *value
	}
	if !medium.IsValidEta(result) {
		return 0, fmt.Errorf("eta_outside must be > 0")
	}
	return result, nil
}

func parseCylindricalGridCutoutSurface(spec *parser.CylindricalGridSurfaceSpec) (bxdf.Scattering, error) {
	lineSurface, err := parseGridLineSurface(spec)
	if err != nil {
		return nil, err
	}
	origin, err := optionalDirection("origin", spec.Origin, maths.NewDirection(0, 0, 0))
	if err != nil {
		return nil, err
	}
	axis, err := optionalDirection("axis", spec.Axis, maths.NewDirection(0, 0, 1))
	if err != nil {
		return nil, err
	}
	if axis.Length() == 0 {
		return nil, fmt.Errorf("axis must have non-zero length")
	}
	referenceAxis, err := optionalDirection("reference_axis", spec.ReferenceAxis, maths.NewDirection(1, 0, 0))
	if err != nil {
		return nil, err
	}
	lineWidth, err := optionalNonNegative("line_width", spec.LineWidth, 0.006)
	if err != nil {
		return nil, err
	}
	gapWidth, err := optionalNonNegative("gap_width", spec.GapWidth, 0.03)
	if err != nil {
		return nil, err
	}
	gapHeight, err := optionalNonNegative("gap_height", spec.GapHeight, 0.03)
	if err != nil {
		return nil, err
	}
	referenceRadius := 1.0
	if spec.ReferenceRadius != nil {
		referenceRadius = *spec.ReferenceRadius
	}
	if referenceRadius <= 0 {
		return nil, fmt.Errorf("reference_radius must be > 0")
	}
	return bsdf.NewCylindricalGridCutout(lineSurface, origin, axis, referenceAxis, gapWidth, gapHeight, lineWidth, referenceRadius), nil
}

func parseGridLineSurface(spec *parser.CylindricalGridSurfaceSpec) (bxdf.Scattering, error) {
	if spec.LineSurface != nil {
		return parseSurface(spec.LineSurface)
	}
	return parseSurface(&parser.SurfaceSpec{Type: parser.SurfaceRoughConductor, Definition: &parser.RoughConductorSurfaceSpec{Eta: json.RawMessage(`[0.15,0.14,0.13]`), K: json.RawMessage(`[4.1,3.5,2.7]`), Weight: json.RawMessage(`[0.88,0.9,0.92]`), Roughness: float64Pointer(0.22)}})
}
func float64Pointer(value float64) *float64 { return &value }

func optionalDirection(name string, values []float64, fallback maths.Direction) (maths.Direction, error) {
	if values == nil {
		return fallback, nil
	}
	if len(values) != 3 {
		return nil, fmt.Errorf("field %q must contain 3 values, got %d", name, len(values))
	}
	for index, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("%s index %d must be finite", name, index)
		}
	}
	return maths.NewDirection(values[0], values[1], values[2]), nil
}
func optionalNonNegative(name string, value *float64, fallback float64) (float64, error) {
	result := fallback
	if value != nil {
		result = *value
	}
	if result < 0 || math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, fmt.Errorf("%s must be >= 0", name)
	}
	return result, nil
}

func parseEmission(spec *parser.EmissionSpec) (emission.Emitter, error) {
	var field emission.RadianceField
	quantity := emission.PeakRadiance
	var err error
	switch definition := spec.Definition.(type) {
	case *parser.ConstantEmissionSpec:
		hasRadiance, hasColor, hasExitance := len(definition.Radiance) > 0, len(definition.Color) > 0, len(definition.Exitance) > 0
		if hasExitance && (hasRadiance || hasColor) {
			return nil, fmt.Errorf("radiance/color and exitance are mutually exclusive")
		}
		var strength optics.SpectralParameter
		if hasExitance {
			strength, err = requiredSpectralParameterRaw(definition.Exitance, "exitance")
			quantity = emission.TotalExitance
		} else if hasRadiance {
			strength, err = requiredSpectralParameterRaw(definition.Radiance, "radiance")
		} else {
			strength, err = requiredSpectralParameterRaw(definition.Color, "color")
		}
		if err != nil {
			return nil, err
		}
		field = emission.Constant{Radiance: strength}
	case *parser.CellPaletteEmissionSpec:
		field, err = parseCellPaletteEmission(definition)
	case *parser.UVKleinEmissionSpec:
		field, err = parseUVKleinEmission(definition)
	default:
		return nil, fmt.Errorf("unsupported emission type %q", spec.Type)
	}
	if err != nil {
		return nil, err
	}
	distribution, err := parseEmissionDistribution(spec.Distribution)
	if err != nil {
		return nil, err
	}
	return emission.NewSurfaceEmitter(field, distribution, quantity), nil
}

func parseEmissionDistribution(spec *parser.EmissionDistributionSpec) (emission.AngularDistribution, error) {
	if spec == nil {
		return emission.NewUniform(emission.TwoSided), nil
	}
	sidedness := emission.FrontSide
	switch spec.Sidedness {
	case "", "front":
	case "back":
		sidedness = emission.BackSide
	case "two_sided":
		sidedness = emission.TwoSided
	default:
		return nil, fmt.Errorf("distribution sidedness must be front, back, or two_sided")
	}
	switch spec.Type {
	case "uniform":
		return emission.NewUniform(sidedness), nil
	case "cosine_power":
		if (spec.Exponent == nil) == (spec.HalfAngleDegrees == nil) {
			return nil, fmt.Errorf("distribution requires exactly one of exponent or half_angle_degrees")
		}
		exponent := 0.0
		if spec.Exponent != nil {
			exponent = *spec.Exponent
		} else {
			angle := *spec.HalfAngleDegrees
			if angle <= 0 || angle > 90 || math.IsNaN(angle) || math.IsInf(angle, 0) {
				return nil, fmt.Errorf("distribution half_angle_degrees must be finite and in (0, 90]")
			}
			if angle != 90 {
				exponent = math.Log(0.5) / math.Log(math.Cos(angle*math.Pi/180))
			}
		}
		result, err := emission.NewCosinePower(exponent, sidedness)
		if err != nil {
			return nil, fmt.Errorf("distribution: %w", err)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported emission distribution type %q", spec.Type)
	}
}

func parseUVKleinEmission(spec *parser.UVKleinEmissionSpec) (emission.RadianceField, error) {
	saturation := 1.0
	if spec.Saturation != nil {
		saturation = *spec.Saturation
	}
	if saturation < 0 || saturation > 1 {
		return nil, fmt.Errorf("saturation must be in [0, 1]")
	}
	lightness := 0.55
	if spec.Lightness != nil {
		lightness = *spec.Lightness
	}
	if lightness < 0 || lightness > 1 {
		return nil, fmt.Errorf("lightness must be in [0, 1]")
	}
	vStripes := 1
	if spec.VStripes != nil {
		vStripes = int(*spec.VStripes)
		if vStripes <= 0 || float64(vStripes) != *spec.VStripes {
			return nil, fmt.Errorf("v_stripes must be a positive integer")
		}
	}
	intensity := 1.0
	if spec.Intensity != nil {
		intensity = *spec.Intensity
	}
	if intensity < 0 {
		return nil, fmt.Errorf("intensity must be >= 0")
	}
	return emission.NewUVKlein(saturation, lightness, vStripes, intensity), nil
}

func parseCellPaletteEmission(spec *parser.CellPaletteEmissionSpec) (emission.RadianceField, error) {
	result := emission.NewCellPalette()
	if spec.Palette != nil {
		if len(spec.Palette) == 0 {
			return nil, fmt.Errorf("palette: must contain at least one color")
		}
		palette := make([]optics.Spectrum, len(spec.Palette))
		for index, values := range spec.Palette {
			if len(values) != 3 {
				return nil, fmt.Errorf("palette[%d]: expected 3 RGB values, got %d", index, len(values))
			}
			if err := utils.ValidateNonNegativeSlice(fmt.Sprintf("palette[%d]", index), values); err != nil {
				return nil, err
			}
			palette[index] = optics.NewSpectrum(values[0], values[1], values[2])
		}
		result.Palette = palette
	}
	intensity := 1.0
	if spec.Intensity != nil {
		intensity = *spec.Intensity
	}
	if intensity < 0 {
		return nil, fmt.Errorf("intensity must be >= 0")
	}
	if intensity != 1 {
		for index := range result.Palette {
			result.Palette[index] = result.Palette[index].MulScalar(intensity)
		}
	}
	switch spec.Shading {
	case "", "solid", "emission":
		result.Shading = emission.CellPaletteShadingEmission
	case "boundary_grid", "grid":
		result.Shading = emission.CellPaletteShadingBoundaryGrid
	default:
		return nil, fmt.Errorf("shading must be \"solid\" or \"boundary_grid\", got %q", spec.Shading)
	}
	if result.Shading == emission.CellPaletteShadingBoundaryGrid {
		if spec.GridColor != nil {
			if len(spec.GridColor) != 3 {
				return nil, fmt.Errorf("grid_color: expected 3 RGB values")
			}
			if err := utils.ValidateNonNegativeSlice("grid_color", spec.GridColor); err != nil {
				return nil, err
			}
			result.GridColor = optics.NewSpectrum(spec.GridColor[0], spec.GridColor[1], spec.GridColor[2])
		}
		thickness, err := optionalNonNegative("grid_thickness", spec.GridThickness, 0.02)
		if err != nil {
			return nil, err
		}
		result.GridThickness = thickness
	}
	return result, nil
}

func parseIORModel(spec *parser.IORSpec, etaInside *float64) (medium.Model, error) {
	if spec != nil {
		switch spec.Type {
		case "constant":
			if spec.Eta == nil {
				return nil, fmt.Errorf("ior: missing required field %q", "eta")
			}
			if !medium.IsValidEta(*spec.Eta) {
				return nil, fmt.Errorf("ior eta must be > 0")
			}
			return medium.NewConstant(*spec.Eta), nil
		case "cauchy":
			if spec.A == nil {
				return nil, fmt.Errorf("ior: missing required field %q", "a")
			}
			if spec.B == nil {
				return nil, fmt.Errorf("ior: missing required field %q", "b")
			}
			c := 0.0
			if spec.C != nil {
				c = *spec.C
			}
			model := medium.NewCauchy(*spec.A, *spec.B, c)
			if !medium.IsValidEta(model.Evaluate(medium.WavelengthMinNM)) || !medium.IsValidEta(model.Evaluate(medium.DefaultWavelengthNM)) || !medium.IsValidEta(model.Evaluate(medium.WavelengthMaxNM)) {
				return nil, fmt.Errorf("ior cauchy coefficients produce invalid eta")
			}
			return model, nil
		default:
			return nil, fmt.Errorf("unsupported ior type %q", spec.Type)
		}
	}
	value := 1.5
	if etaInside != nil {
		value = *etaInside
	}
	if !medium.IsValidEta(value) {
		return nil, fmt.Errorf("eta_inside must be > 0")
	}
	return medium.NewConstant(value), nil
}

func requiredSpectralParameterRaw(raw json.RawMessage, key string) (optics.SpectralParameter, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("missing required field %q", key)
	}
	return parseSpectralParameterRaw(raw, key)
}
func optionalSpectralParameterRaw(raw json.RawMessage, key string, fallback optics.SpectralParameter) (optics.SpectralParameter, error) {
	if len(raw) == 0 {
		return fallback, nil
	}
	return parseSpectralParameterRaw(raw, key)
}
func parseSpectralParameterRaw(raw json.RawMessage, key string) (optics.SpectralParameter, error) {
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	parameter, err := parseSpectralParameterValue(key, value)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	return parameter, nil
}

func parseSpectralParameterValue(key string, value interface{}) (optics.SpectralParameter, error) {
	if mapped, ok := value.(map[string]interface{}); ok {
		return parseSpectralParameterObject(mapped)
	}
	values, err := utils.ToFloat64Slice(value)
	if err != nil {
		return nil, err
	}
	if err := utils.RequireSliceLength(key, values, 3); err != nil {
		return nil, err
	}
	if err := utils.ValidateNonNegativeSlice("legacy rgb", values); err != nil {
		return nil, err
	}
	return spectrum_parameter.NewRGBParameter(optics.NewSpectrum(values[0], values[1], values[2])), nil
}

// Media is still a map-based protocol and shares the polymorphic spectral leaf parser.
func optionalSpectralParameterField(data map[string]interface{}, key string, fallback optics.SpectralParameter) (optics.SpectralParameter, bool, error) {
	value, ok := data[key]
	if !ok {
		return fallback, false, nil
	}
	parameter, err := parseSpectralParameterValue(key, value)
	if err != nil {
		return nil, true, fmt.Errorf("field %q: %w", key, err)
	}
	return parameter, true, nil
}

func parseSpectralParameterObject(def map[string]interface{}) (optics.SpectralParameter, error) {
	parameterType, err := utils.RequiredStringField(def, "type")
	if err != nil {
		return nil, err
	}
	switch parameterType {
	case "rgb":
		values, err := utils.RequiredFloat64SliceField(def, "value", 3)
		if err != nil {
			return nil, err
		}
		if err := utils.ValidateNonNegativeSlice("value", values); err != nil {
			return nil, err
		}
		space, ok, err := utils.OptionalStringField(def, "space")
		if err != nil {
			return nil, err
		}
		if !ok {
			space = string(optics.RGBColorSpaceLinearSRGB)
		}
		value := optics.NewSpectrum(values[0], values[1], values[2])
		switch optics.RGBColorSpace(space) {
		case optics.RGBColorSpaceLinearSRGB:
			return spectrum_parameter.NewRGBParameter(value), nil
		case optics.RGBColorSpaceSRGB:
			return spectrum_parameter.NewSRGBParameter(value), nil
		case optics.RGBColorSpaceACEScg:
			return spectrum_parameter.NewACEScgParameter(value), nil
		default:
			return nil, fmt.Errorf("unsupported rgb color space %q", space)
		}
	case "constant":
		value, err := utils.RequiredFloat64Field(def, "value")
		if err != nil {
			return nil, err
		}
		if value < 0 {
			return nil, fmt.Errorf("value must be >= 0")
		}
		return spectrum_parameter.NewConstantParameter(value), nil
	case "sampled":
		wavelengths, err := utils.RequiredFloat64SliceField(def, "wavelengths_nm")
		if err != nil {
			return nil, err
		}
		values, err := utils.RequiredFloat64SliceField(def, "values")
		if err != nil {
			return nil, err
		}
		if len(wavelengths) != len(values) {
			return nil, fmt.Errorf("wavelengths_nm and values must have the same length")
		}
		if len(wavelengths) < 2 {
			return nil, fmt.Errorf("sampled spectrum must contain at least 2 samples")
		}
		if err := utils.ValidateStrictlyIncreasing("wavelengths_nm", wavelengths); err != nil {
			return nil, err
		}
		if err := utils.ValidateNonNegativeSlice("values", values); err != nil {
			return nil, err
		}
		interpolation, ok, err := utils.OptionalStringField(def, "interpolation")
		if err != nil {
			return nil, err
		}
		if ok && interpolation != "linear" {
			return nil, fmt.Errorf("unsupported interpolation %q", interpolation)
		}
		return spectrum_parameter.NewSampledParameter(wavelengths, values), nil
	case "blackbody":
		temperature, err := utils.RequiredFloat64Field(def, "temperature")
		if err != nil {
			return nil, err
		}
		if temperature <= 0 {
			return nil, fmt.Errorf("temperature must be > 0")
		}
		scale, ok, err := utils.OptionalFloat64Field(def, "scale")
		if err != nil {
			return nil, err
		}
		if !ok {
			scale = 1
		}
		if scale < 0 {
			return nil, fmt.Errorf("scale must be >= 0")
		}
		return spectrum_parameter.NewBlackbodyParameter(temperature, scale), nil
	default:
		return nil, fmt.Errorf("unsupported spectral parameter type %q", parameterType)
	}
}
