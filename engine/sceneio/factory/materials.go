package factory

import (
	"errors"
	"fmt"

	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/material/emission"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
)

func ParseMaterials(script *Script) (map[string]*material.Material, error) {
	if script == nil {
		return nil, errors.New("script is nil")
	}

	materials := make(map[string]*material.Material, len(script.Materials))
	var parseErrors []error

	for idx, matDef := range script.Materials {
		context := fmt.Sprintf("material[%d]", idx)

		id, err := requiredStringField(matDef, "id")
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
				SpectrumMode: bxdf.SpectrumRGB,
			},
		}

		if surfaceDef, ok, err := optionalMapField(matDef, "surface"); err != nil {
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

		if emissionDef, ok, err := optionalMapField(matDef, "emission"); err != nil {
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
	surfaceType, err := requiredStringField(def, "type")
	if err != nil {
		return nil, err
	}

	switch surfaceType {
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
		etaOutside, ok, err := optionalFloat64Field(def, "eta_outside")
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
		roughness, ok, err := optionalFloat64Field(def, "roughness")
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
		return bsdf.NewSingle(bxdf.NewRoughConductorParameter(eta, k, alpha)), nil
	default:
		return nil, fmt.Errorf("unsupported surface type %q", surfaceType)
	}
}

func parseEmission(def map[string]interface{}) (material.Emitter, error) {
	emissionType, err := requiredStringField(def, "type")
	if err != nil {
		return nil, err
	}

	switch emissionType {
	case "constant":
		radiance, err := requiredEmissionRadianceField(def)
		if err != nil {
			return nil, err
		}
		return emission.NewConstantParameter(radiance), nil
	default:
		return nil, fmt.Errorf("unsupported emission type %q", emissionType)
	}
}

func parseIORModel(def map[string]interface{}) (medium.Model, error) {
	if iorDef, ok, err := optionalMapField(def, "ior"); err != nil {
		return nil, err
	} else if ok {
		iorType, err := requiredStringField(iorDef, "type")
		if err != nil {
			return nil, fmt.Errorf("ior: %w", err)
		}
		switch iorType {
		case "constant":
			eta, err := requiredFloat64Field(iorDef, "eta")
			if err != nil {
				return nil, fmt.Errorf("ior: %w", err)
			}
			if !medium.IsValidEta(eta) {
				return nil, fmt.Errorf("ior eta must be > 0")
			}
			return medium.NewConstant(eta), nil
		case "cauchy":
			a, err := requiredFloat64Field(iorDef, "a")
			if err != nil {
				return nil, fmt.Errorf("ior: %w", err)
			}
			b, err := requiredFloat64Field(iorDef, "b")
			if err != nil {
				return nil, fmt.Errorf("ior: %w", err)
			}
			c, ok, err := optionalFloat64Field(iorDef, "c")
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

	etaInside, ok, err := optionalFloat64Field(def, "eta_inside")
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

	values, err := toFloat64Slice(value)
	if err != nil {
		return nil, err
	}
	if err := requireSliceLength(key, values, 3); err != nil {
		return nil, err
	}
	if err := validateNonNegativeSlice("legacy rgb", values); err != nil {
		return nil, err
	}
	return spectrum_parameter.NewRGBParameter(optics.NewSpectrum(values[0], values[1], values[2])), nil
}

func parseSpectralParameterObject(def map[string]interface{}) (optics.SpectralParameter, error) {
	parameterType, err := requiredStringField(def, "type")
	if err != nil {
		return nil, err
	}

	switch parameterType {
	case "rgb":
		values, err := requiredFloat64SliceField(def, "value", 3)
		if err != nil {
			return nil, err
		}
		if err := validateNonNegativeSlice("value", values); err != nil {
			return nil, err
		}
		space, ok, err := optionalStringField(def, "space")
		if err != nil {
			return nil, err
		}
		if !ok {
			space = string(optics.ColorSpaceLinearSRGB)
		}
		value := optics.NewSpectrum(values[0], values[1], values[2])
		switch optics.ColorSpace(space) {
		case optics.ColorSpaceLinearSRGB:
			return spectrum_parameter.NewRGBParameter(value), nil
		case optics.ColorSpaceSRGB:
			return spectrum_parameter.NewSRGBParameter(value), nil
		case optics.ColorSpaceACEScg:
			return spectrum_parameter.NewACEScgParameter(value), nil
		default:
			return nil, fmt.Errorf("unsupported rgb color space %q", space)
		}
	case "constant":
		value, err := requiredFloat64Field(def, "value")
		if err != nil {
			return nil, err
		}
		if value < 0 {
			return nil, fmt.Errorf("value must be >= 0")
		}
		return spectrum_parameter.NewConstantParameter(value), nil
	case "sampled":
		wavelengths, err := requiredFloat64SliceField(def, "wavelengths_nm")
		if err != nil {
			return nil, err
		}
		values, err := requiredFloat64SliceField(def, "values")
		if err != nil {
			return nil, err
		}
		if len(wavelengths) != len(values) {
			return nil, fmt.Errorf("wavelengths_nm and values must have the same length")
		}
		if len(wavelengths) < 2 {
			return nil, fmt.Errorf("sampled spectrum must contain at least 2 samples")
		}
		if err := validateStrictlyIncreasing("wavelengths_nm", wavelengths); err != nil {
			return nil, err
		}
		if err := validateNonNegativeSlice("values", values); err != nil {
			return nil, err
		}
		interpolation, ok, err := optionalStringField(def, "interpolation")
		if err != nil {
			return nil, err
		}
		if ok && interpolation != "linear" {
			return nil, fmt.Errorf("unsupported interpolation %q", interpolation)
		}
		return spectrum_parameter.NewSampledParameter(wavelengths, values), nil
	case "blackbody":
		temperature, err := requiredFloat64Field(def, "temperature")
		if err != nil {
			return nil, err
		}
		if temperature <= 0 {
			return nil, fmt.Errorf("temperature must be > 0")
		}
		scale, ok, err := optionalFloat64Field(def, "scale")
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

func validateNonNegativeSlice(name string, values []float64) error {
	for i, value := range values {
		if value < 0 {
			return fmt.Errorf("%s index %d must be >= 0", name, i)
		}
	}
	return nil
}

func validateStrictlyIncreasing(name string, values []float64) error {
	for i := 1; i < len(values); i++ {
		if values[i] <= values[i-1] {
			return fmt.Errorf("%s must be strictly increasing", name)
		}
	}
	return nil
}

func optionalMapField(data map[string]interface{}, key string) (map[string]interface{}, bool, error) {
	value, ok := data[key]
	if !ok {
		return nil, false, nil
	}
	mapped, ok := value.(map[string]interface{})
	if !ok {
		return nil, true, fmt.Errorf("field %q: expected object, got %T", key, value)
	}
	return mapped, true, nil
}
