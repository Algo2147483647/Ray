package app

import (
	"runtime"
	"testing"

	"github.com/Algo2147483647/ray/engine/go/internal/controller"
)

func TestResolveRenderConfigMergesScriptAndCLI(t *testing.T) {
	script := &controller.Script{
		Render: controller.RenderScript{
			Samples:      12,
			ThreadNum:    3,
			CameraIndex:  1,
			Width:        800,
			Height:       600,
			OutputImage:  "scene.png",
			OutputFilm:   "scene.bin",
			ResumeFilm:   "resume.bin",
			DebugOutput:  "scene-debug.json",
			Exposure:     1.5,
			ToneMapping:  "reinhard",
			Gamma:        2.2,
			SpectrumMode: "rgb",
		},
	}

	config := ResolveRenderConfig(script, RenderOverrides{
		ScriptPath:   "custom.json",
		CameraIndex:  2,
		ThreadNum:    6,
		Width:        1024,
		Samples:      32,
		OutputImage:  "override.png",
		Exposure:     0.75,
		ToneMapping:  "aces",
		Gamma:        1.8,
		SpectrumMode: "hero_wavelength",
	})

	if config.ScriptPath != "custom.json" {
		t.Fatalf("unexpected script path: %s", config.ScriptPath)
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
	if config.OutputFilm != "scene.bin" || config.ResumeFilm != "resume.bin" || config.DebugOutput != "scene-debug.json" {
		t.Fatalf("unexpected output fallback: film=%s resume=%s debug=%s", config.OutputFilm, config.ResumeFilm, config.DebugOutput)
	}
	if config.Exposure != 0.75 || config.ToneMapping != "aces" || config.Gamma != 1.8 {
		t.Fatalf("unexpected output transform: exposure=%f tone=%s gamma=%f", config.Exposure, config.ToneMapping, config.Gamma)
	}
	if config.SpectrumMode != "hero_wavelength" {
		t.Fatalf("expected CLI spectrum mode override, got %s", config.SpectrumMode)
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
