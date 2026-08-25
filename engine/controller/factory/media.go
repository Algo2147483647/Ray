package factory

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/utils"

	"github.com/Algo2147483647/ray/engine/model/material/medium"
)

func ParseMediaRegistry(script *parser.Script) (*medium.Registry, error) {
	registry := medium.NewRegistry()
	if script == nil || len(script.Media) == 0 {
		return registry, nil
	}

	for name, def := range script.Media {
		context := fmt.Sprintf("medium %q", name)
		mediumType, ok, err := utils.OptionalStringField(def, "type")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
		if !ok {
			mediumType = "homogeneous"
		}
		if mediumType != "homogeneous" {
			return nil, fmt.Errorf("%s: unsupported medium type %q", context, mediumType)
		}
		if _, exists := def["sigma_s"]; exists {
			return nil, fmt.Errorf("%s: sigma_s is unsupported until volume scattering is implemented", context)
		}

		etaModel, err := parseMediumIORModel(def)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
		sigmaA, _, err := optionalSpectralParameterField(def, "sigma_a", nil)
		if err != nil {
			return nil, fmt.Errorf("%s sigma_a: %w", context, err)
		}
		if _, err := registry.RegisterHomogeneousWithCoefficients(
			name,
			etaModel,
			spectralCoefficient{parameter: sigmaA},
		); err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
	}

	return registry, nil
}

func parseMediumBoundary(spec *parser.MediumBoundarySpec, registry *medium.Registry) (medium.Boundary, error) {
	if spec == nil {
		return medium.Boundary{}, nil
	}
	outsideName := spec.Outside
	if outsideName == "" {
		outsideName = "air"
	}
	outside, ok := registry.ID(outsideName)
	if !ok {
		return medium.Boundary{}, fmt.Errorf("unknown outside medium %q", outsideName)
	}

	insideName := spec.Inside
	if insideName == "" {
		return medium.Boundary{}, fmt.Errorf("missing required field %q", "inside")
	}
	inside, ok := registry.ID(insideName)
	if !ok {
		return medium.Boundary{}, fmt.Errorf("unknown inside medium %q", insideName)
	}

	priority := 0
	if spec.Priority != nil {
		priority = *spec.Priority
	}

	return medium.Boundary{
		Outside:  outside,
		Inside:   inside,
		Priority: priority,
		Thin:     spec.Thin,
	}, nil
}

func parseMediumIORModel(def map[string]interface{}) (medium.Model, error) {
	iorDef, ok, err := utils.OptionalMapField(def, "ior")
	if err != nil {
		return nil, err
	}
	if !ok {
		return medium.NewConstant(1), nil
	}

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

type spectralCoefficient struct {
	parameter optics.SpectralParameter
}

func (c spectralCoefficient) Eval(ctx medium.WavelengthContext) medium.CoefficientSpectrum {
	if c.parameter == nil {
		return medium.CoefficientSpectrum{}
	}
	evaluated := c.parameter.Eval(ctx)
	if evaluated.HasSamples() {
		return medium.NewSampledCoefficientSpectrum(evaluated.Samples)
	}
	return medium.NewRGBCoefficientSpectrum(
		evaluated.RGBChannel(0),
		evaluated.RGBChannel(1),
		evaluated.RGBChannel(2),
	)
}
