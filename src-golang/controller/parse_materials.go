package controller

import (
	"errors"
	"fmt"
	"gonum.org/v1/gonum/mat"
	"src-golang/model/optics"
)

func ParseMaterials(script *Script) (map[string]*optics.Material, error) {
	if script == nil {
		return nil, errors.New("script is nil")
	}

	materials := make(map[string]*optics.Material, len(script.Materials))
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

		color, err := requiredFloat64SliceField(matDef, "color", 3)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		}

		material := optics.NewMaterial(mat.NewVecDense(len(color), color))

		if value, ok, err := optionalBoolField(matDef, "radiate"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			material.Radiation = value
		}

		if value, ok, err := optionalStringField(matDef, "radiation_type"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			material.RadiationType = value
		}

		if value, ok, err := optionalFloat64Field(matDef, "reflectivity"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			material.Reflectivity = value
		}

		if value, ok, err := optionalFloat64Field(matDef, "refractivity"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			material.Refractivity = value
		}

		if value, exists := matDef["refractive_index"]; exists {
			switch typed := value.(type) {
			case []interface{}, []float64:
				values, err := toFloat64Slice(typed)
				if err != nil {
					parseErrors = append(parseErrors, fmt.Errorf("%s: field %q: %w", context, "refractive_index", err))
					continue
				}
				if err := requireSliceLength("refractive_index", values, 1, 3); err != nil {
					parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
					continue
				}
				material.RefractiveIndex = mat.NewVecDense(len(values), values)
			default:
				number, err := toFloat64(value)
				if err != nil {
					parseErrors = append(parseErrors, fmt.Errorf("%s: field %q: %w", context, "refractive_index", err))
					continue
				}
				material.RefractiveIndex = mat.NewVecDense(1, []float64{number})
			}
		}

		if value, ok, err := optionalFloat64Field(matDef, "diffuse_loss"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			material.DiffuseLoss = value
		}

		if value, ok, err := optionalFloat64Field(matDef, "reflect_loss"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			material.ReflectLoss = value
		}

		if value, ok, err := optionalFloat64Field(matDef, "refract_loss"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			material.RefractLoss = value
		}

		if value, ok, err := optionalStringField(matDef, "color_func"); err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s: %w", context, err))
			continue
		} else if ok {
			colorFunc, exists := optics.ColorFuncMap[value]
			if !exists {
				parseErrors = append(parseErrors, fmt.Errorf("%s: unknown color_func %q", context, value))
				continue
			}
			material.ColorFunc = colorFunc
		}

		materials[id] = material
	}

	if len(parseErrors) > 0 {
		return nil, errors.Join(parseErrors...)
	}

	return materials, nil
}
