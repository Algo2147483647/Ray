package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
	"github.com/Algo2147483647/ray/engine/go/internal/material/ior"
	"github.com/Algo2147483647/ray/engine/go/internal/material/medium"
)

func ParseMediaRegistry(script *Script) (*medium.Registry, error) {
	registry := medium.NewRegistry()
	if script == nil || len(script.Media) == 0 {
		return registry, nil
	}

	for name, def := range script.Media {
		context := fmt.Sprintf("medium %q", name)
		mediumType, ok, err := optionalStringField(def, "type")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
		if !ok {
			mediumType = "homogeneous"
		}
		if mediumType != "homogeneous" {
			return nil, fmt.Errorf("%s: unsupported medium type %q", context, mediumType)
		}

		etaModel, err := parseMediumIORModel(def)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
		sigmaA, _, err := optionalSpectralParameterField(def, "sigma_a", core.NewConstantParameter(0))
		if err != nil {
			return nil, fmt.Errorf("%s sigma_a: %w", context, err)
		}
		sigmaS, _, err := optionalSpectralParameterField(def, "sigma_s", core.NewConstantParameter(0))
		if err != nil {
			return nil, fmt.Errorf("%s sigma_s: %w", context, err)
		}

		if _, err := registry.RegisterHomogeneous(name, etaModel, sigmaA, sigmaS); err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
	}

	return registry, nil
}

func parseMediumBoundary(def map[string]interface{}, registry *medium.Registry) (medium.Boundary, error) {
	boundaryDef, ok, err := optionalMapField(def, "medium_boundary")
	if err != nil || !ok {
		return medium.Boundary{}, err
	}

	outsideName, ok, err := optionalStringField(boundaryDef, "outside")
	if err != nil {
		return medium.Boundary{}, err
	}
	if !ok {
		outsideName = "air"
	}
	outside, ok := registry.ID(outsideName)
	if !ok {
		return medium.Boundary{}, fmt.Errorf("unknown outside medium %q", outsideName)
	}

	insideName, err := requiredStringField(boundaryDef, "inside")
	if err != nil {
		return medium.Boundary{}, err
	}
	inside, ok := registry.ID(insideName)
	if !ok {
		return medium.Boundary{}, fmt.Errorf("unknown inside medium %q", insideName)
	}

	priorityValue, ok, err := optionalFloat64Field(boundaryDef, "priority")
	if err != nil {
		return medium.Boundary{}, err
	}
	priority := 0
	if ok {
		priority = int(priorityValue)
		if float64(priority) != priorityValue {
			return medium.Boundary{}, fmt.Errorf("priority must be an integer")
		}
	}

	thin, ok, err := optionalBoolField(boundaryDef, "thin")
	if err != nil {
		return medium.Boundary{}, err
	}
	if !ok {
		thin = false
	}

	return medium.Boundary{
		Outside:  outside,
		Inside:   inside,
		Priority: priority,
		Thin:     thin,
	}, nil
}

func parseMediumIORModel(def map[string]interface{}) (ior.Model, error) {
	iorDef, ok, err := optionalMapField(def, "ior")
	if err != nil {
		return nil, err
	}
	if !ok {
		return ior.NewConstant(1), nil
	}

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
		if !ior.IsValidEta(eta) {
			return nil, fmt.Errorf("ior eta must be > 0")
		}
		return ior.NewConstant(eta), nil
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
		model := ior.NewCauchy(a, b, c)
		if !ior.IsValidEta(model.Evaluate(ior.WavelengthMinNM)) ||
			!ior.IsValidEta(model.Evaluate(ior.DefaultWavelengthNM)) ||
			!ior.IsValidEta(model.Evaluate(ior.WavelengthMaxNM)) {
			return nil, fmt.Errorf("ior cauchy coefficients produce invalid eta")
		}
		return model, nil
	default:
		return nil, fmt.Errorf("unsupported ior type %q", iorType)
	}
}
