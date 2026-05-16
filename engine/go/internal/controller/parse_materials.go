package controller

import (
	"errors"
	"fmt"

	"github.com/Algo2147483647/ray/engine/go/internal/material/bsdf"
	"github.com/Algo2147483647/ray/engine/go/internal/material/bxdf"
	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
	"github.com/Algo2147483647/ray/engine/go/internal/material/emission"
)

func ParseMaterials(script *Script) (map[string]*core.Material, error) {
	if script == nil {
		return nil, errors.New("script is nil")
	}

	materials := make(map[string]*core.Material, len(script.Materials))
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

		material := &core.Material{
			Metadata: core.MaterialMetadata{
				Name:         id,
				SpectrumMode: core.SpectrumRGB,
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

func parseSurface(def map[string]interface{}) (core.BSDF, error) {
	surfaceType, err := requiredStringField(def, "type")
	if err != nil {
		return nil, err
	}

	switch surfaceType {
	case "lambert":
		albedo, err := requiredSpectrumField(def, "albedo")
		if err != nil {
			return nil, err
		}
		return bsdf.NewSingle(bxdf.NewLambert(albedo)), nil
	case "specular_reflection":
		reflectance, err := optionalSpectrumField(def, "reflectance", core.ConstantSpectrum(1))
		if err != nil {
			return nil, err
		}
		return bsdf.NewSingle(bxdf.NewSpecularReflection(reflectance)), nil
	case "specular_dielectric":
		reflectance, err := optionalSpectrumField(def, "reflectance", core.ConstantSpectrum(1))
		if err != nil {
			return nil, err
		}
		transmittance, err := optionalSpectrumField(def, "transmittance", core.ConstantSpectrum(1))
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
		etaInside, ok, err := optionalFloat64Field(def, "eta_inside")
		if err != nil {
			return nil, err
		}
		if !ok {
			etaInside = 1.5
		}
		if etaOutside <= 0 || etaInside <= 0 {
			return nil, fmt.Errorf("eta values must be > 0")
		}
		return bsdf.NewSingle(bxdf.NewSpecularDielectric(reflectance, transmittance, etaOutside, etaInside)), nil
	case "rough_conductor":
		eta, err := requiredSpectrumField(def, "eta")
		if err != nil {
			return nil, err
		}
		k, err := requiredSpectrumField(def, "k")
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
		return bsdf.NewSingle(bxdf.NewRoughConductor(eta, k, alpha)), nil
	default:
		return nil, fmt.Errorf("unsupported surface type %q", surfaceType)
	}
}

func parseEmission(def map[string]interface{}) (core.Emitter, error) {
	emissionType, err := requiredStringField(def, "type")
	if err != nil {
		return nil, err
	}

	switch emissionType {
	case "constant":
		color, err := requiredSpectrumField(def, "color")
		if err != nil {
			return nil, err
		}
		return emission.NewConstant(color), nil
	default:
		return nil, fmt.Errorf("unsupported emission type %q", emissionType)
	}
}

func requiredSpectrumField(data map[string]interface{}, key string) (core.Spectrum, error) {
	values, err := requiredFloat64SliceField(data, key, 3)
	if err != nil {
		return core.Spectrum{}, err
	}
	for i, value := range values {
		if value < 0 {
			return core.Spectrum{}, fmt.Errorf("field %q index %d must be >= 0", key, i)
		}
	}
	return core.NewSpectrum(values[0], values[1], values[2]), nil
}

func optionalSpectrumField(data map[string]interface{}, key string, fallback core.Spectrum) (core.Spectrum, error) {
	values, ok, err := optionalFloat64SliceField(data, key, 3)
	if err != nil {
		return core.Spectrum{}, err
	}
	if !ok {
		return fallback, nil
	}
	for i, value := range values {
		if value < 0 {
			return core.Spectrum{}, fmt.Errorf("field %q index %d must be >= 0", key, i)
		}
	}
	return core.NewSpectrum(values[0], values[1], values[2]), nil
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
