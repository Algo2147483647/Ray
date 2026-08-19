package factory

import (
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

	for idx, matDef := range script.Materials {
		context := fmt.Sprintf("material[%d]", idx)

		id, err := utils.RequiredStringField(matDef, "id")
		if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		}
		context = fmt.Sprintf("material[%d] id=%q", idx, id)

		if _, exists := materials[id]; exists {
			parseErrors = append(parseErrors, fmt.Errorf("%s: duplicate material id", context))
			continue
		}

		material := &material.Material{
			Metadata: material.MaterialMetadata{
				Name:         id,
				SpectrumMode: optics.SpectrumModeRGB,
			},
		}

		if surfaceDef, ok, err := utils.OptionalMapField(matDef, "surface"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			surface, err := parseSurface(surfaceDef)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("%s surface: %w", context, err))
				continue
			}
			material.Surface = surface
		}

		if emissionDef, ok, err := utils.OptionalMapField(matDef, "emission"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			emitter, err := parseEmission(emissionDef)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("%s emission: %w", context, err))
				continue
			}
			material.Emission = emitter
		}

		if !material.HasSurface() && !material.HasEmission() {
			parseErrors = append(parseErrors, fmt.Errorf("%s: material requires surface or emission", context))
			continue
		}

		materials[id] = material
	}

	if len(parseErrors) > 0 {
		return nil, errors.Join(parseErrors...)
	}

	return materials, nil
}

func parseSurface(def map[string]interface{}) (bsdf.BSDF, error) {
	surfaceType, err := utils.RequiredStringField(def, "type")
	if err != nil {
		return nil, err
	}

	switch surfaceType {
	case "weighted_mixture":
		componentsRaw, ok := def["components"]
		if !ok {
			return nil, fmt.Errorf("missing required field %q", "components")
		}
		componentDefs, ok := componentsRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("field %q: expected array, got %T", "components", componentsRaw)
		}
		if len(componentDefs) == 0 {
			return nil, fmt.Errorf("field %q must not be empty", "components")
		}

		components := make([]bsdf.WeightedBxDF, 0, len(componentDefs))
		for index, componentRaw := range componentDefs {
			component, ok := componentRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("components[%d]: expected object, got %T", index, componentRaw)
			}
			weight, err := utils.RequiredFloat64Field(component, "weight")
			if err != nil {
				return nil, fmt.Errorf("components[%d]: %w", index, err)
			}
			if weight <= 0 || math.IsNaN(weight) || math.IsInf(weight, 0) {
				return nil, fmt.Errorf("components[%d] weight must be finite and > 0", index)
			}
			surfaceDef, ok, err := utils.OptionalMapField(component, "surface")
			if err != nil {
				return nil, fmt.Errorf("components[%d]: %w", index, err)
			}
			if !ok {
				return nil, fmt.Errorf("components[%d]: missing required field %q", index, "surface")
			}
			surface, err := parseSurface(surfaceDef)
			if err != nil {
				return nil, fmt.Errorf("components[%d] surface: %w", index, err)
			}
			components = append(components, bsdf.WeightedBxDF{Weight: weight, BxDF: surface})
		}
		return bsdf.NewWeightedMixture(components...), nil

	case "lambert":
		albedo, err := requiredSpectralParameterField(def, "albedo")
		if err != nil {
			return nil, err
		}
		return bsdf.NewSingle(bxdf.NewLambertParameter(albedo)), nil

	case "specular_reflection":
		reflectance, _, err := optionalSpectralParameterField(def, "reflectance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		return bsdf.NewSingle(bxdf.NewSpecularReflectionParameter(reflectance)), nil

	case "specular_dielectric":
		reflectance, _, err := optionalSpectralParameterField(def, "reflectance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		transmittance, _, err := optionalSpectralParameterField(def, "transmittance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		etaOutside, ok, err := utils.OptionalFloat64Field(def, "eta_outside")
		if err != nil {
			return nil, err
		}
		if !ok {
			etaOutside = 1
		}
		if !medium.IsValidEta(etaOutside) {
			return nil, fmt.Errorf("eta_outside must be > 0")
		}
		insideIOR, err := parseIORModel(def)
		if err != nil {
			return nil, err
		}
		return bsdf.NewSingle(bxdf.NewSpecularDielectricParameter(reflectance, transmittance, etaOutside, insideIOR)), nil

	case "rough_conductor":
		eta, err := requiredSpectralParameterField(def, "eta")
		if err != nil {
			return nil, err
		}
		k, err := requiredSpectralParameterField(def, "k")
		if err != nil {
			return nil, err
		}
		roughness, ok, err := utils.OptionalFloat64Field(def, "roughness")
		if err != nil {
			return nil, err
		}
		if !ok {
			roughness = 0.25
		}
		if roughness < 0 || roughness > 1 {
			return nil, fmt.Errorf("roughness must be in [0, 1]")
		}
		alpha := roughness * roughness
		weight, _, err := optionalSpectralParameterField(def, "weight", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		conductor := bxdf.NewRoughConductorParameter(eta, k, alpha)
		conductor.Weight = weight
		return bsdf.NewSingle(conductor), nil

	case "rough_dielectric_reflection":
		reflectance, _, err := optionalSpectralParameterField(def, "reflectance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		etaOutside, ok, err := utils.OptionalFloat64Field(def, "eta_outside")
		if err != nil {
			return nil, err
		}
		if !ok {
			etaOutside = 1
		}
		if !medium.IsValidEta(etaOutside) {
			return nil, fmt.Errorf("eta_outside must be > 0")
		}
		insideIOR, err := parseIORModel(def)
		if err != nil {
			return nil, err
		}
		roughness, ok, err := utils.OptionalFloat64Field(def, "roughness")
		if err != nil {
			return nil, err
		}
		if !ok {
			roughness = 0.25
		}
		if roughness < 0 || roughness > 1 {
			return nil, fmt.Errorf("roughness must be in [0, 1]")
		}
		return bsdf.NewSingle(bxdf.NewRoughDielectricReflectionParameter(
			reflectance,
			etaOutside,
			insideIOR,
			roughness*roughness,
		)), nil

	case "cylindrical_grid_cutout", "wire_mesh":
		return parseCylindricalGridCutoutSurface(def)

	case "rough_dielectric_transmission":
		transmittance, _, err := optionalSpectralParameterField(def, "transmittance", spectrum_parameter.NewConstantParameter(1))
		if err != nil {
			return nil, err
		}
		etaOutside, ok, err := utils.OptionalFloat64Field(def, "eta_outside")
		if err != nil {
			return nil, err
		}
		if !ok {
			etaOutside = 1
		}
		if !medium.IsValidEta(etaOutside) {
			return nil, fmt.Errorf("eta_outside must be > 0")
		}
		insideIOR, err := parseIORModel(def)
		if err != nil {
			return nil, err
		}
		roughness, ok, err := utils.OptionalFloat64Field(def, "roughness")
		if err != nil {
			return nil, err
		}
		if !ok {
			roughness = 0.25
		}
		if roughness < 0 || roughness > 1 {
			return nil, fmt.Errorf("roughness must be in [0, 1]")
		}
		alpha := roughness * roughness
		return bsdf.NewSingle(bxdf.NewRoughDielectricTransmissionParameter(transmittance, etaOutside, insideIOR, alpha)), nil

	default:
		return nil, fmt.Errorf("unsupported surface type %q", surfaceType)
	}
}

func parseCylindricalGridCutoutSurface(def map[string]interface{}) (bsdf.BSDF, error) {
	lineSurface, err := parseGridLineSurface(def)
	if err != nil {
		return nil, err
	}

	origin, err := optionalDirectionField(def, "origin", maths.NewDirection(0, 0, 0))
	if err != nil {
		return nil, err
	}
	axis, err := optionalDirectionField(def, "axis", maths.NewDirection(0, 0, 1))
	if err != nil {
		return nil, err
	}
	if axis.Length() == 0 {
		return nil, fmt.Errorf("axis must have non-zero length")
	}
	referenceAxis, err := optionalDirectionField(def, "reference_axis", maths.NewDirection(1, 0, 0))
	if err != nil {
		return nil, err
	}

	lineWidth, ok, err := utils.OptionalFloat64Field(def, "line_width")
	if err != nil {
		return nil, err
	}
	if !ok {
		lineWidth = 0.006
	}
	if lineWidth < 0 {
		return nil, fmt.Errorf("line_width must be >= 0")
	}

	gapWidth, ok, err := utils.OptionalFloat64Field(def, "gap_width")
	if err != nil {
		return nil, err
	}
	if !ok {
		gapWidth = 0.03
	}
	if gapWidth < 0 {
		return nil, fmt.Errorf("gap_width must be >= 0")
	}

	gapHeight, ok, err := utils.OptionalFloat64Field(def, "gap_height")
	if err != nil {
		return nil, err
	}
	if !ok {
		gapHeight = 0.03
	}
	if gapHeight < 0 {
		return nil, fmt.Errorf("gap_height must be >= 0")
	}

	referenceRadius, ok, err := utils.OptionalFloat64Field(def, "reference_radius")
	if err != nil {
		return nil, err
	}
	if !ok {
		referenceRadius = 1
	}
	if referenceRadius <= 0 {
		return nil, fmt.Errorf("reference_radius must be > 0")
	}

	return bsdf.NewCylindricalGridCutout(
		lineSurface,
		origin,
		axis,
		referenceAxis,
		gapWidth,
		gapHeight,
		lineWidth,
		referenceRadius,
	), nil
}

func parseGridLineSurface(def map[string]interface{}) (bsdf.BSDF, error) {
	lineSurfaceDef, ok, err := utils.OptionalMapField(def, "line_surface")
	if err != nil {
		return nil, err
	}
	if !ok {
		lineSurfaceDef = map[string]interface{}{
			"type":      "rough_conductor",
			"eta":       []interface{}{0.15, 0.14, 0.13},
			"k":         []interface{}{4.1, 3.5, 2.7},
			"roughness": 0.22,
			"weight":    []interface{}{0.88, 0.9, 0.92},
		}
	}
	return parseSurface(lineSurfaceDef)
}

func optionalDirectionField(def map[string]interface{}, key string, fallback maths.Direction) (maths.Direction, error) {
	values, ok, err := utils.OptionalFloat64SliceField(def, key, 3)
	if err != nil {
		return nil, err
	}
	if !ok {
		return fallback, nil
	}
	for i, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("%s index %d must be finite", key, i)
		}
	}
	return maths.NewDirection(values[0], values[1], values[2]), nil
}

func parseEmission(def map[string]interface{}) (emission.Emitter, error) {
	emissionType, err := utils.RequiredStringField(def, "type")
	if err != nil {
		return nil, err
	}

	var field emission.RadianceField
	quantity := emission.PeakRadiance
	switch emissionType {
	case "constant":
		_, hasRadiance := def["radiance"]
		_, hasColor := def["color"]
		_, hasExitance := def["exitance"]
		if hasExitance && (hasRadiance || hasColor) {
			return nil, fmt.Errorf("radiance/color and exitance are mutually exclusive")
		}
		var strength optics.SpectralParameter
		if hasExitance {
			strength, err = requiredSpectralParameterField(def, "exitance")
			quantity = emission.TotalExitance
		} else {
			strength, err = requiredEmissionRadianceField(def)
		}
		if err != nil {
			return nil, err
		}
		field = emission.Constant{Radiance: strength}
	case "cell_palette":
		field, err = parseCellPaletteEmission(def)
	case "uv_klein":
		field, err = parseUVKleinEmission(def)
	default:
		return nil, fmt.Errorf("unsupported emission type %q", emissionType)
	}
	if err != nil {
		return nil, err
	}
	distribution, err := parseEmissionDistribution(def)
	if err != nil {
		return nil, err
	}
	return emission.NewSurfaceEmitter(field, distribution, quantity), nil
}

func parseEmissionDistribution(def map[string]interface{}) (emission.AngularDistribution, error) {
	distributionDef, ok, err := utils.OptionalMapField(def, "distribution")
	if err != nil {
		return nil, err
	}
	if !ok {
		return emission.NewUniform(emission.TwoSided), nil
	}

	distributionType, err := utils.RequiredStringField(distributionDef, "type")
	if err != nil {
		return nil, fmt.Errorf("distribution: %w", err)
	}
	sidedness := emission.FrontSide
	if side, ok, err := utils.OptionalStringField(distributionDef, "sidedness"); err != nil {
		return nil, fmt.Errorf("distribution: %w", err)
	} else if ok {
		switch side {
		case "front":
			sidedness = emission.FrontSide
		case "back":
			sidedness = emission.BackSide
		case "two_sided":
			sidedness = emission.TwoSided
		default:
			return nil, fmt.Errorf("distribution sidedness must be front, back, or two_sided")
		}
	}

	switch distributionType {
	case "uniform":
		return emission.NewUniform(sidedness), nil
	case "cosine_power":
		exponent, hasExponent, err := utils.OptionalFloat64Field(distributionDef, "exponent")
		if err != nil {
			return nil, fmt.Errorf("distribution: %w", err)
		}
		halfAngle, hasHalfAngle, err := utils.OptionalFloat64Field(distributionDef, "half_angle_degrees")
		if err != nil {
			return nil, fmt.Errorf("distribution: %w", err)
		}
		if hasExponent == hasHalfAngle {
			return nil, fmt.Errorf("distribution requires exactly one of exponent or half_angle_degrees")
		}
		if hasHalfAngle {
			if halfAngle <= 0 || halfAngle > 90 || math.IsNaN(halfAngle) || math.IsInf(halfAngle, 0) {
				return nil, fmt.Errorf("distribution half_angle_degrees must be finite and in (0, 90]")
			}
			if halfAngle == 90 {
				exponent = 0
			} else {
				exponent = math.Log(0.5) / math.Log(math.Cos(halfAngle*math.Pi/180))
			}
		}
		result, err := emission.NewCosinePower(exponent, sidedness)
		if err != nil {
			return nil, fmt.Errorf("distribution: %w", err)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported emission distribution type %q", distributionType)
	}
}

func parseUVKleinEmission(def map[string]interface{}) (emission.RadianceField, error) {
	saturation, ok, err := utils.OptionalFloat64Field(def, "saturation")
	if err != nil {
		return nil, err
	}
	if !ok {
		saturation = 1
	}
	if saturation < 0 || saturation > 1 {
		return nil, fmt.Errorf("saturation must be in [0, 1]")
	}

	lightness, ok, err := utils.OptionalFloat64Field(def, "lightness")
	if err != nil {
		return nil, err
	}
	if !ok {
		lightness = 0.55
	}
	if lightness < 0 || lightness > 1 {
		return nil, fmt.Errorf("lightness must be in [0, 1]")
	}

	vStripeValue, ok, err := utils.OptionalFloat64Field(def, "v_stripes")
	if err != nil {
		return nil, err
	}
	vStripes := int(vStripeValue)
	if !ok {
		vStripes = 1
	}
	if vStripes <= 0 || float64(vStripes) != vStripeValue {
		return nil, fmt.Errorf("v_stripes must be a positive integer")
	}

	intensity, ok, err := utils.OptionalFloat64Field(def, "intensity")
	if err != nil {
		return nil, err
	}
	if !ok {
		intensity = 1
	}
	if intensity < 0 {
		return nil, fmt.Errorf("intensity must be >= 0")
	}

	return emission.NewUVKlein(saturation, lightness, vStripes, intensity), nil
}

// parseCellPaletteEmission builds a CellPalette debug emitter from a scene
// definition. All fields are optional:
//
//   - "palette":         array of N RGB triples (defaults to DefaultCellPalette;
//     cells beyond palette length wrap modulo).
//   - "intensity":       scalar applied to every palette entry.
//   - "shading":         "solid" (default) or "boundary_grid".
//   - "grid_color":      RGB triple for boundary stripes (defaults to white).
//   - "grid_thickness":  world-space half-width of the grid in scene units
//     (defaults to 0.02 of the smallest object extent at parse
//     time — but here we just default to a small absolute value
//     of 0.02 and rely on the user to tune).
func parseCellPaletteEmission(def map[string]interface{}) (emission.RadianceField, error) {
	cp := emission.NewCellPalette()

	if rawPalette, ok := def["palette"]; ok {
		entries, ok := rawPalette.([]interface{})
		if !ok {
			return nil, fmt.Errorf("palette: expected an array of RGB triples")
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("palette: must contain at least one color")
		}
		palette := make([]optics.Spectrum, 0, len(entries))
		for i, entry := range entries {
			values, err := utils.ToFloat64Slice(entry)
			if err != nil {
				return nil, fmt.Errorf("palette[%d]: %w", i, err)
			}
			if len(values) != 3 {
				return nil, fmt.Errorf("palette[%d]: expected 3 RGB values, got %d", i, len(values))
			}
			if err := utils.ValidateNonNegativeSlice(fmt.Sprintf("palette[%d]", i), values); err != nil {
				return nil, err
			}
			palette = append(palette, optics.NewSpectrum(values[0], values[1], values[2]))
		}
		cp.Palette = palette
	}

	if intensity, ok, err := utils.OptionalFloat64Field(def, "intensity"); err != nil {
		return nil, err
	} else if ok {
		if intensity < 0 {
			return nil, fmt.Errorf("intensity must be >= 0")
		}
		for i := range cp.Palette {
			cp.Palette[i] = cp.Palette[i].MulScalar(intensity)
		}
	}

	if shading, ok, err := utils.OptionalStringField(def, "shading"); err != nil {
		return nil, err
	} else if ok {
		switch shading {
		case "solid", "emission", "":
			cp.Shading = emission.CellPaletteShadingEmission
		case "boundary_grid", "grid":
			cp.Shading = emission.CellPaletteShadingBoundaryGrid
		default:
			return nil, fmt.Errorf("shading must be \"solid\" or \"boundary_grid\", got %q", shading)
		}
	}

	if cp.Shading == emission.CellPaletteShadingBoundaryGrid {
		if rawGrid, ok := def["grid_color"]; ok {
			values, err := utils.ToFloat64Slice(rawGrid)
			if err != nil {
				return nil, fmt.Errorf("grid_color: %w", err)
			}
			if len(values) != 3 {
				return nil, fmt.Errorf("grid_color: expected 3 RGB values")
			}
			if err := utils.ValidateNonNegativeSlice("grid_color", values); err != nil {
				return nil, err
			}
			cp.GridColor = optics.NewSpectrum(values[0], values[1], values[2])
		}

		thickness, ok, err := utils.OptionalFloat64Field(def, "grid_thickness")
		if err != nil {
			return nil, err
		}
		if !ok {
			thickness = 0.02
		}
		if thickness < 0 {
			return nil, fmt.Errorf("grid_thickness must be >= 0")
		}
		cp.GridThickness = thickness
	}

	return cp, nil

}

func parseIORModel(def map[string]interface{}) (medium.Model, error) {
	if iorDef, ok, err := utils.OptionalMapField(def, "ior"); err != nil {
		return nil, err
	} else if ok {
		iorType, err := utils.RequiredStringField(iorDef, "type")
		if err != nil {
			return nil, fmt.Errorf("ior: %w", err)
		}
		switch iorType {
		case "constant":
			eta, err := utils.RequiredFloat64Field(iorDef, "eta")
			if err != nil {
				return nil, fmt.Errorf("ior: %w", err)
			}
			if !medium.IsValidEta(eta) {
				return nil, fmt.Errorf("ior eta must be > 0")
			}
			return medium.NewConstant(eta), nil
		case "cauchy":
			a, err := utils.RequiredFloat64Field(iorDef, "a")
			if err != nil {
				return nil, fmt.Errorf("ior: %w", err)
			}
			b, err := utils.RequiredFloat64Field(iorDef, "b")
			if err != nil {
				return nil, fmt.Errorf("ior: %w", err)
			}
			c, ok, err := utils.OptionalFloat64Field(iorDef, "c")
			if err != nil {
				return nil, fmt.Errorf("ior: %w", err)
			}
			if !ok {
				c = 0
			}
			model := medium.NewCauchy(a, b, c)
			if !medium.IsValidEta(model.Evaluate(medium.WavelengthMinNM)) ||
				!medium.IsValidEta(model.Evaluate(medium.DefaultWavelengthNM)) ||
				!medium.IsValidEta(model.Evaluate(medium.WavelengthMaxNM)) {
				return nil, fmt.Errorf("ior cauchy coefficients produce invalid eta")
			}
			return model, nil
		default:
			return nil, fmt.Errorf("unsupported ior type %q", iorType)
		}
	}

	etaInside, ok, err := utils.OptionalFloat64Field(def, "eta_inside")
	if err != nil {
		return nil, err
	}
	if !ok {
		etaInside = 1.5
	}
	if !medium.IsValidEta(etaInside) {
		return nil, fmt.Errorf("eta_inside must be > 0")
	}
	return medium.NewConstant(etaInside), nil
}

func requiredEmissionRadianceField(data map[string]interface{}) (optics.SpectralParameter, error) {
	if _, ok := data["radiance"]; ok {
		return requiredSpectralParameterField(data, "radiance")
	}
	return requiredSpectralParameterField(data, "color")
}

func requiredSpectralParameterField(data map[string]interface{}, key string) (optics.SpectralParameter, error) {
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("missing required field %q", key)
	}
	parameter, err := parseSpectralParameterValue(key, value)
	if err != nil {
		return nil, fmt.Errorf("field %q: %w", key, err)
	}
	return parameter, nil
}

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
