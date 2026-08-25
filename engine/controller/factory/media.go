package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

func ParseMediaRegistry(script *parser.Script) (*medium.Registry, error) {
	registry := medium.NewRegistry()
	if script == nil || len(script.Media) == 0 {
		return registry, nil
	}

	for name, spec := range script.Media {
		context := fmt.Sprintf("medium %q", name)
		mediumType := spec.Type
		if mediumType == "" {
			mediumType = "homogeneous"
		}
		if mediumType != "homogeneous" {
			return nil, fmt.Errorf("%s: unsupported medium type %q", context, mediumType)
		}
		if len(spec.SigmaS) != 0 {
			return nil, fmt.Errorf("%s: sigma_s is unsupported until volume scattering is implemented", context)
		}

		defaultEta := 1.0
		etaModel, err := parseIORModel(spec.IOR, &defaultEta)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
		sigmaA, err := optionalSpectralParameterRaw(spec.SigmaA, "sigma_a", nil)
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

type spectralCoefficient struct {
	parameter optics.SpectralParameter
}

func (c spectralCoefficient) Eval(ctx medium.WavelengthContext) medium.CoefficientSpectrum {
	if c.parameter == nil {
		return medium.CoefficientSpectrum{}
	}
	evaluated := c.parameter.Eval(ctx)
	if evaluated.HasSamples() {
		samples := make([]float64, evaluated.SampleCount())
		for i := range samples {
			samples[i] = evaluated.Sample(i)
		}
		return medium.NewSampledCoefficientSpectrum(samples)
	}
	return medium.NewRGBCoefficientSpectrum(
		evaluated.RGBChannel(0),
		evaluated.RGBChannel(1),
		evaluated.RGBChannel(2),
	)
}
