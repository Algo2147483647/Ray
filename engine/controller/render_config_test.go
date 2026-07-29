package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"gonum.org/v1/gonum/mat"
)

func TestResolveRenderConfigAcceptsFilmColorSpaceAlias(t *testing.T) {
	config := ResolveRenderConfig(&parser.Script{
		Render: parser.RenderScript{
			FilmColorSpace: "acescg",
		},
	}, RenderOverrides{CameraIndex: -1})

	if config.ColorSpace != "acescg" {
		t.Fatalf("expected working_space alias to set color space, got %q", config.ColorSpace)
	}
}

func TestResolveRenderConfigPrefersColorSpaceOverAlias(t *testing.T) {
	config := ResolveRenderConfig(&parser.Script{
		Render: parser.RenderScript{
			ColorSpace:     "xyz",
			FilmColorSpace: "acescg",
		},
	}, RenderOverrides{CameraIndex: -1})

	if config.ColorSpace != "xyz" {
		t.Fatalf("expected color_space to win over working_space alias, got %q", config.ColorSpace)
	}
}

func TestParseRenderOverridesRejectsRepeatedScripts(t *testing.T) {
	_, err := ParseRenderOverrides([]string{
		"--script", "studio.json",
		"--script", "geometry.json",
	})
	if err == nil {
		t.Fatal("expected repeated engine scripts to fail")
	}
}

func TestParseRenderOverridesAcceptsBDPT(t *testing.T) {
	overrides, err := ParseRenderOverrides([]string{"--integrator", "bdpt"})
	if err != nil {
		t.Fatalf("parse bdpt integrator: %v", err)
	}
	if overrides.Integrator != "bdpt" {
		t.Fatalf("integrator = %q, want bdpt", overrides.Integrator)
	}
}

func TestParseRenderOverridesRejectsUnknownIntegrator(t *testing.T) {
	if _, err := ParseRenderOverrides([]string{"--integrator", "magic"}); err == nil {
		t.Fatal("expected unknown integrator to fail")
	}
}

func TestParseRenderOverridesRejectsMultiplePositionalScripts(t *testing.T) {
	_, err := ParseRenderOverrides([]string{"studio.json", "geometry.json"})
	if err == nil {
		t.Fatal("expected multiple positional engine scripts to fail")
	}
}

func TestParseRenderOverridesRejectsResumeFilm(t *testing.T) {
	_, err := ParseRenderOverrides([]string{"--resume-film", "existing.bin"})
	if err == nil {
		t.Fatal("expected engine resume-film flag to fail; studio owns film resume")
	}
}

func TestParseRenderOverridesAcceptsPixelWindows(t *testing.T) {
	overrides, err := ParseRenderOverrides([]string{
		"--pixel-window", "100-150,600-650",
		"--pixel-window", "2:4,6:8",
	})
	if err != nil {
		t.Fatalf("parse overrides: %v", err)
	}

	if len(overrides.PixelWindows) != 2 {
		t.Fatalf("expected two pixel windows, got %d", len(overrides.PixelWindows))
	}
	assertIntSlice(t, overrides.PixelWindows[0].Min, []int{100, 600})
	assertIntSlice(t, overrides.PixelWindows[0].Max, []int{150, 650})
	assertIntSlice(t, overrides.PixelWindows[1].Min, []int{2, 6})
	assertIntSlice(t, overrides.PixelWindows[1].Max, []int{4, 8})
}

func TestParseRenderOverridesRejectsInvalidPixelWindow(t *testing.T) {
	_, err := ParseRenderOverrides([]string{"--pixel-window", "10:10,0:1"})
	if err == nil {
		t.Fatal("expected invalid pixel window to fail")
	}
}

func TestResolveRenderConfigsExpandsRenderJobs(t *testing.T) {
	configs := ResolveRenderConfigs(&parser.Script{
		Render: parser.RenderScript{
			Samples:    8,
			Width:      320,
			OutputFilm: "base.bin",
		},
		Renders: []parser.RenderScript{
			{OutputFilm: "front.bin"},
			{Samples: 32, OutputFilm: "detail.bin"},
		},
	}, RenderOverrides{CameraIndex: -1})

	if len(configs) != 2 {
		t.Fatalf("expected two render configs, got %d", len(configs))
	}
	if configs[0].Samples != 8 || configs[0].Width != 320 || configs[0].OutputFilm != "front.bin" {
		t.Fatalf("unexpected first render config: %+v", configs[0])
	}
	if configs[1].Samples != 32 || configs[1].Width != 320 || configs[1].OutputFilm != "detail.bin" {
		t.Fatalf("unexpected second render config: %+v", configs[1])
	}
}

func TestResolveRenderConfigPrefersOverridePixelWindows(t *testing.T) {
	config := ResolveRenderConfig(&parser.Script{
		Render: parser.RenderScript{
			PixelWindows: []camera.PixelWindow{
				{Min: []int{1, 1}, Max: []int{2, 2}},
			},
		},
	}, RenderOverrides{
		CameraIndex: -1,
		PixelWindows: []camera.PixelWindow{
			{Min: []int{3, 3}, Max: []int{4, 4}},
		},
	})

	if len(config.PixelWindows) != 1 {
		t.Fatalf("expected one pixel window, got %d", len(config.PixelWindows))
	}
	assertIntSlice(t, config.PixelWindows[0].Min, []int{3, 3})
	assertIntSlice(t, config.PixelWindows[0].Max, []int{4, 4})
}

func TestResolveRenderConfigsRenderJobInheritsCameraIndexWhenOmitted(t *testing.T) {
	configs := ResolveRenderConfigs(&parser.Script{
		Render: parser.RenderScript{
			CameraIndex:    2,
			CameraIndexSet: true,
		},
		Renders: []parser.RenderScript{
			{OutputFilm: "inherited.bin"},
			{CameraIndex: 0, CameraIndexSet: true, OutputFilm: "override.bin"},
		},
	}, RenderOverrides{CameraIndex: -1})

	if configs[0].CameraIndex != 2 {
		t.Fatalf("expected first render job to inherit camera index 2, got %d", configs[0].CameraIndex)
	}
	if configs[1].CameraIndex != 0 {
		t.Fatalf("expected second render job to override camera index to 0, got %d", configs[1].CameraIndex)
	}
}

func TestSelectRenderCameraAppliesOverridesToHyperbolicCamera(t *testing.T) {
	cam := &camera.HyperbolicCamera{Camera3D: camera.Camera3D{
		Position:     mat.NewVecDense(3, []float64{0, 0, 0}),
		Direction:    mat.NewVecDense(3, []float64{1, 0, 0}),
		Up:           mat.NewVecDense(3, []float64{0, 0, 1}),
		Width:        400,
		Height:       400,
		FieldOfViews: []float64{70, 70},
	}}
	h := NewHandler()
	h.Scene.Cameras = []camera.Camera{cam}

	_, shape, err := h.selectRenderCamera(0, 120, 80)
	if err != nil {
		t.Fatalf("select camera: %v", err)
	}
	if cam.Width != 120 || cam.Height != 80 || shape[0] != 120 || shape[1] != 80 {
		t.Fatalf("expected hyperbolic camera dimensions to update, got camera=%dx%d shape=%v", cam.Width, cam.Height, shape)
	}
}

func TestSelectRenderCameraRequiresCamera(t *testing.T) {
	h := NewHandler()
	_, _, err := h.selectRenderCamera(0, 120, 80)
	if err == nil {
		t.Fatal("expected selecting without cameras to fail")
	}
}

func TestSelectRenderCameraAppliesOverridesToSphericalCamera(t *testing.T) {
	cam := &camera.SphericalCamera{
		Position:     mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		Forward:      mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		Up:           mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		Width:        400,
		Height:       400,
		FieldOfViews: []float64{70, 70},
	}
	h := NewHandler()
	h.Scene.Cameras = []camera.Camera{cam}

	_, shape, err := h.selectRenderCamera(0, 160, 100)
	if err != nil {
		t.Fatalf("select camera: %v", err)
	}
	if cam.Width != 160 || cam.Height != 100 || shape[0] != 160 || shape[1] != 100 {
		t.Fatalf("expected spherical camera dimensions to update, got camera=%dx%d shape=%v", cam.Width, cam.Height, shape)
	}
}

func TestConfigureRenderConfigRejectsOutOfBoundsPixelWindow(t *testing.T) {
	cam := &camera.SphericalCamera{
		Position:     mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		Forward:      mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		Up:           mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		Width:        400,
		Height:       400,
		FieldOfViews: []float64{70, 70},
	}
	h := NewHandler()
	h.Scene.Cameras = []camera.Camera{cam}

	h.ConfigureRenderConfig(RenderConfig{
		CameraIndex:  0,
		Width:        10,
		Height:       10,
		PixelWindows: []camera.PixelWindow{{Min: []int{9, 0}, Max: []int{11, 1}}},
	})

	if h.err == nil {
		t.Fatal("expected out-of-bounds pixel window to fail")
	}
}

func TestConfigureRenderConfigRejectsBDPTInCurvedGeometry(t *testing.T) {
	h := NewHandler()
	h.Scene.Geometry = geometry.Klein()
	h.ConfigureRenderConfig(RenderConfig{
		Integrator:  "bdpt",
		Dimension:   3,
		CameraIndex: 0,
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
