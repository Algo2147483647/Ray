package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
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

func TestParseRenderOverridesRejectsMultiplePositionalScripts(t *testing.T) {
	_, err := ParseRenderOverrides([]string{"studio.json", "geometry.json"})
	if err == nil {
		t.Fatal("expected multiple positional engine scripts to fail")
	}
}

func TestResolveRenderConfigsExpandsRenderJobs(t *testing.T) {
	configs := ResolveRenderConfigs(&parser.Script{
		Render: parser.RenderScript{
			Samples:     8,
			Width:       320,
			OutputImage: "base.png",
		},
		Renders: []parser.RenderScript{
			{OutputImage: "front.png"},
			{Samples: 32, OutputImage: "detail.png"},
		},
	}, RenderOverrides{CameraIndex: -1})

	if len(configs) != 2 {
		t.Fatalf("expected two render configs, got %d", len(configs))
	}
	if configs[0].Samples != 8 || configs[0].Width != 320 || configs[0].OutputImage != "front.png" {
		t.Fatalf("unexpected first render config: %+v", configs[0])
	}
	if configs[1].Samples != 32 || configs[1].Width != 320 || configs[1].OutputImage != "detail.png" {
		t.Fatalf("unexpected second render config: %+v", configs[1])
	}
}

func TestResolveRenderConfigsRenderJobInheritsCameraIndexWhenOmitted(t *testing.T) {
	configs := ResolveRenderConfigs(&parser.Script{
		Render: parser.RenderScript{
			CameraIndex:    2,
			CameraIndexSet: true,
		},
		Renders: []parser.RenderScript{
			{OutputImage: "inherited.png"},
			{CameraIndex: 0, CameraIndexSet: true, OutputImage: "override.png"},
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
		Position:    mat.NewVecDense(3, []float64{0, 0, 0}),
		Direction:   mat.NewVecDense(3, []float64{1, 0, 0}),
		Up:          mat.NewVecDense(3, []float64{0, 0, 1}),
		Width:       400,
		Height:      400,
		FieldOfView: 70,
		AspectRatio: 1,
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
		Position:    mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		Forward:     mat.NewVecDense(4, []float64{0, 1, 0, 0}),
		Up:          mat.NewVecDense(4, []float64{0, 0, 1, 0}),
		Width:       400,
		Height:      400,
		FieldOfView: 70,
		AspectRatio: 1,
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
