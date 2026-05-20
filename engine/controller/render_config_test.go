package controller

import (
	"runtime"
	"testing"
)

func TestResolveRenderConfigMergesScriptAndCLI(t *testing.T) {
	script := &Script{
		Render: RenderScript{
			Dimension:         4,
			Samples:           12,
			ThreadNum:         3,
			CameraIndex:       1,
			Width:             800,
			Height:            600,
			OutputImage:       "scene.png",
			OutputFilm:        "scene.bin",
			ResumeFilm:        "resume.bin",
			Exposure:          1.5,
			ToneMapping:       "reinhard",
			Gamma:             2.2,
			SpectrumMode:      "rgb",
			WavelengthSamples: 2,
			ColorSpace:        "linear_srgb",
		},
	}

	config := ResolveRenderConfig(script, RenderOverrides{
		ScriptPath:        "custom.json",
		CameraIndex:       2,
		ThreadNum:         6,
		Width:             1024,
		Samples:           32,
		OutputImage:       "override.png",
		Exposure:          0.75,
		ToneMapping:       "aces",
		Gamma:             1.8,
		SpectrumMode:      "hero_wavelength",
		WavelengthSamples: 3,
		ColorSpace:        "linear_srgb",
	})

	if config.ScriptPath != "custom.json" {
		t.Fatalf("unexpected script path: %s", config.ScriptPath)
	}
	if config.Dimension != 4 {
		t.Fatalf("unexpected dimension: %d", config.Dimension)
	}
	if config.CameraIndex != 2 {
		t.Fatalf("expected CLI camera index override, got %d", config.CameraIndex)
	}
	if config.ThreadNum != 6 {
		t.Fatalf("expected CLI thread override, got %d", config.ThreadNum)
	}
	if config.Width != 1024 || config.Height != 600 {
		t.Fatalf("unexpected dimensions: %dx%d", config.Width, config.Height)
	}
	if config.Samples != 32 {
		t.Fatalf("expected CLI samples override, got %d", config.Samples)
	}
	if config.OutputImage != "override.png" {
		t.Fatalf("expected CLI output image override, got %s", config.OutputImage)
	}
	if config.OutputFilm != "scene.bin" || config.ResumeFilm != "resume.bin" {
		t.Fatalf("unexpected output fallback: film=%s resume=%s", config.OutputFilm, config.ResumeFilm)
	}
	if config.Exposure != 0.75 || config.ToneMapping != "aces" || config.Gamma != 1.8 {
		t.Fatalf("unexpected output transform: exposure=%f tone=%s gamma=%f", config.Exposure, config.ToneMapping, config.Gamma)
	}
	if config.SpectrumMode != "hero_wavelength" {
		t.Fatalf("expected CLI spectrum mode override, got %s", config.SpectrumMode)
	}
	if config.WavelengthSamples != 3 {
		t.Fatalf("expected CLI wavelength samples override, got %d", config.WavelengthSamples)
	}
	if config.ColorSpace != "linear_srgb" {
		t.Fatalf("unexpected working space: %s", config.ColorSpace)
	}
}

func TestResolveRenderConfigDefaultsThreadNumToNumCPU(t *testing.T) {
	config := ResolveRenderConfig(nil, RenderOverrides{ScriptPath: defaultScriptPath})
	if config.ThreadNum != runtime.NumCPU() {
		t.Fatalf("expected default thread count %d, got %d", runtime.NumCPU(), config.ThreadNum)
	}
	if config.Exposure != 1 || config.ToneMapping != "linear" || config.Gamma != 1 {
		t.Fatalf("unexpected default output transform: exposure=%f tone=%s gamma=%f", config.Exposure, config.ToneMapping, config.Gamma)
	}
	if config.SpectrumMode != "hero_wavelength" {
		t.Fatalf("unexpected default spectrum mode: %s", config.SpectrumMode)
	}
	if config.WavelengthSamples != 1 {
		t.Fatalf("unexpected default wavelength samples: %d", config.WavelengthSamples)
	}
	if config.ColorSpace != "linear_srgb" {
		t.Fatalf("unexpected default working space: %s", config.ColorSpace)
	}
}

func TestResolveRenderConfigDefaultsSampledModeToMultipleWavelengths(t *testing.T) {
	config := ResolveRenderConfig(&Script{
		Render: RenderScript{
			SpectrumMode: "sampled",
		},
	}, RenderOverrides{ScriptPath: defaultScriptPath})

	if config.WavelengthSamples != 4 {
		t.Fatalf("expected sampled mode to default to 4 wavelength samples, got %d", config.WavelengthSamples)
	}
}

func TestParseRenderOverridesRejectsUnsupportedToneMapping(t *testing.T) {
	if _, err := ParseRenderOverrides([]string{"--tone-mapping", "magic"}); err == nil {
		t.Fatal("expected unsupported tone mapping to fail")
	}
}

func TestParseRenderOverridesRejectsUnsupportedSpectrumMode(t *testing.T) {
	if _, err := ParseRenderOverrides([]string{"--spectrum-mode", "magic"}); err == nil {
		t.Fatal("expected unsupported spectrum mode to fail")
	}
}

func TestParseRenderOverridesRejectsNegativeWavelengthSamples(t *testing.T) {
	if _, err := ParseRenderOverrides([]string{"--wavelength-samples", "-1"}); err == nil {
		t.Fatal("expected negative wavelength samples to fail")
	}
}

func TestParseRenderOverridesAcceptsXYZColorSpace(t *testing.T) {
	overrides, err := ParseRenderOverrides([]string{"--working-space", "xyz"})
	if err != nil {
		t.Fatalf("expected xyz working space to be supported: %v", err)
	}
	if overrides.ColorSpace != "xyz" {
		t.Fatalf("unexpected working space: %s", overrides.ColorSpace)
	}
}

func TestParseRenderOverridesRejectsUnsupportedColorSpace(t *testing.T) {
	if _, err := ParseRenderOverrides([]string{"--working-space", "magic"}); err == nil {
		t.Fatal("expected unsupported working space to fail")
	}
}
