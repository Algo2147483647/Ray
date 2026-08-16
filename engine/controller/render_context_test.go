package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"gonum.org/v1/gonum/mat"
)

func TestParseArgsRejectsRepeatedScripts(t *testing.T) {
	h := NewHandler().ParseArgs([]string{
		"--script", "studio.json",
		"--script", "geometry.json",
	})
	if h.err == nil {
		t.Fatal("expected repeated engine scripts to fail")
	}
}

func TestParseArgsAcceptsOnlyScriptPath(t *testing.T) {
	h := NewHandler().ParseArgs([]string{"--script", "scene.json"})
	if h.err != nil {
		t.Fatalf("parse script path: %v", h.err)
	}
	if h.ScriptPath != "scene.json" {
		t.Fatalf("script path = %q, want scene.json", h.ScriptPath)
	}
}

func TestParseArgsRejectsMultiplePositionalScripts(t *testing.T) {
	h := NewHandler().ParseArgs([]string{"studio.json", "geometry.json"})
	if h.err == nil {
		t.Fatal("expected multiple positional engine scripts to fail")
	}
}

func TestParseArgsRejectsRenderOverrides(t *testing.T) {
	for _, args := range [][]string{
		{"--integrator", "bdpt"},
		{"--camera-id", "main"},
		{"--dimension", "3"},
		{"--samples", "8"},
		{"--threads", "4"},
		{"--widths", "16,16"},
		{"--output-film", "image.bin"},
		{"--spectrum-mode", "sampled"},
		{"--wavelength-samples", "4"},
		{"--pixel-window", "0:1,0:1"},
	} {
		if h := NewHandler().ParseArgs(args); h.err == nil {
			t.Fatalf("expected Engine to reject render override %v", args)
		}
	}
}

func TestResolveRenderContextsExpandsRenderJobs(t *testing.T) {
	contexts := ResolveRenderContexts(&parser.Script{
		Renders: []parser.RenderScript{
			{Samples: 8, CameraID: "front"},
			{Samples: 32, CameraID: "detail"},
		},
		Cameras: []parser.CameraScript{
			{ID: "front", Film: &camera.Film{Shape: []int{320, 200}, OutputFilm: "front.bin"}},
			{ID: "detail", Film: &camera.Film{Shape: []int{640, 400}, OutputFilm: "detail.bin"}},
		},
	})

	if len(contexts) != 2 {
		t.Fatalf("expected two render contexts, got %d", len(contexts))
	}
	if contexts[0].Samples != 8 || contexts[0].CameraID != "front" {
		t.Fatalf("unexpected first render context: %+v", contexts[0])
	}
	if contexts[1].Samples != 32 || contexts[1].CameraID != "detail" {
		t.Fatalf("unexpected second render context: %+v", contexts[1])
	}
}

func TestResolveRenderContextsUsesEachJobCameraID(t *testing.T) {
	contexts := ResolveRenderContexts(&parser.Script{
		Renders: []parser.RenderScript{
			{CameraID: "inherited"},
			{CameraID: "side"},
		},
		Cameras: []parser.CameraScript{
			{ID: "side", Film: &camera.Film{Shape: []int{10, 10}}},
			{ID: "unused", Film: &camera.Film{Shape: []int{10, 10}}},
			{ID: "inherited", Film: &camera.Film{Shape: []int{10, 10}}},
		},
	})

	if contexts[0].CameraID != "inherited" {
		t.Fatalf("expected first render job camera ID, got %q", contexts[0].CameraID)
	}
	if contexts[1].CameraID != "side" {
		t.Fatalf("expected second render job camera ID, got %q", contexts[1].CameraID)
	}
}

func TestResolveRenderContextsUsesDefaultsForOmittedJobValues(t *testing.T) {
	contexts := ResolveRenderContexts(&parser.Script{
		Renders: []parser.RenderScript{
			{Samples: 32, CameraID: "detail"},
			{CameraID: "main"},
		},
	})

	if contexts[1].Samples != defaultSamples || contexts[1].CameraID != "main" {
		t.Fatalf("second render inherited values from the first: %+v", contexts[1])
	}
}

func TestResolveRenderContextsDoesNotNormalizeStudioConfiguration(t *testing.T) {
	contexts := ResolveRenderContexts(&parser.Script{Renders: []parser.RenderScript{{
		SpectrumMode:      "sampled",
		WavelengthSamples: 1,
	}}})

	if contexts[0].WavelengthSamples != 1 {
		t.Fatalf("Engine normalized wavelength samples to %d, want canonical JSON value 1", contexts[0].WavelengthSamples)
	}
}

func TestConfigureRenderContextRejectsOutOfBoundsPixelWindow(t *testing.T) {
	cam := &camera.SphericalCamera{
		Camera:       camera.Camera{Film: camera.NewFilm(10, 10)},
		Position:     mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		Coordinates:  []*mat.VecDense{mat.NewVecDense(4, []float64{0, 1, 0, 0}), mat.NewVecDense(4, []float64{0, 0, 0, 1}), mat.NewVecDense(4, []float64{0, 0, 1, 0})},
		FieldOfViews: []float64{70, 70},
	}
	h := NewHandler()
	cam.Film.PixelWindows = []camera.PixelWindow{{Min: []int{9, 0}, Max: []int{11, 1}}}
	h.Scene.Cameras = map[string]camera.RayCamera{"main": cam}

	h.ConfigureRenderContext(RenderContext{
		CameraID: "main",
	})

	if h.err == nil {
		t.Fatal("expected out-of-bounds pixel window to fail")
	}
}

func TestConfigureRenderContextRejectsBDPTInCurvedGeometry(t *testing.T) {
	h := NewHandler()
	h.Scene.Geometry = geometry.Klein()
	h.ConfigureRenderContext(RenderContext{
		Integrator: "bdpt",
		Dimension:  3,
		CameraID:   "missing",
	})
	if h.err == nil {
		t.Fatal("expected curved-space BDPT to fail during configuration")
	}
}

func assertIntSlice(t *testing.T, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
