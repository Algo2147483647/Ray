package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"gonum.org/v1/gonum/mat"
)

func TestParseRenderArgsRejectsRGBFilmMode(t *testing.T) {
	if _, err := parseRenderArgs([]string{"--spectrum-mode", "rgb"}); err == nil {
		t.Fatal("expected Engine to reject RGB Film output mode")
	}
}

func TestParseRenderArgsRejectsRepeatedScripts(t *testing.T) {
	_, err := parseRenderArgs([]string{
		"--script", "studio.json",
		"--script", "geometry.json",
	})
	if err == nil {
		t.Fatal("expected repeated engine scripts to fail")
	}
}

func TestParseRenderArgsAcceptsBDPT(t *testing.T) {
	context, err := parseRenderArgs([]string{"--integrator", "bdpt"})
	if err != nil {
		t.Fatalf("parse bdpt integrator: %v", err)
	}
	if context.Integrator != "bdpt" {
		t.Fatalf("integrator = %q, want bdpt", context.Integrator)
	}
}

func TestParseRenderArgsAcceptsLightTracingNames(t *testing.T) {
	for _, name := range []string{"light_tracing", "light_trace"} {
		context, err := parseRenderArgs([]string{"--integrator", name})
		if err != nil {
			t.Fatalf("parse %q integrator: %v", name, err)
		}
		if context.Integrator != name {
			t.Fatalf("integrator = %q, want %q", context.Integrator, name)
		}
	}
}

func TestParseRenderArgsRejectsUnknownIntegrator(t *testing.T) {
	if _, err := parseRenderArgs([]string{"--integrator", "magic"}); err == nil {
		t.Fatal("expected unknown integrator to fail")
	}
}

func TestParseRenderArgsRejectsMultiplePositionalScripts(t *testing.T) {
	_, err := parseRenderArgs([]string{"studio.json", "geometry.json"})
	if err == nil {
		t.Fatal("expected multiple positional engine scripts to fail")
	}
}

func TestParseRenderArgsRejectsResumeFilm(t *testing.T) {
	_, err := parseRenderArgs([]string{"--resume-film", "existing.bin"})
	if err == nil {
		t.Fatal("expected engine resume-film flag to fail; studio owns film resume")
	}
}

func TestParseRenderArgsAcceptsPixelWindows(t *testing.T) {
	context, err := parseRenderArgs([]string{
		"--pixel-window", "100-150,600-650",
		"--pixel-window", "2:4,6:8",
	})
	if err != nil {
		t.Fatalf("parse render arguments: %v", err)
	}

	if len(context.PixelWindows) != 2 {
		t.Fatalf("expected two pixel windows, got %d", len(context.PixelWindows))
	}
	assertIntSlice(t, context.PixelWindows[0].Min, []int{100, 600})
	assertIntSlice(t, context.PixelWindows[0].Max, []int{150, 650})
	assertIntSlice(t, context.PixelWindows[1].Min, []int{2, 6})
	assertIntSlice(t, context.PixelWindows[1].Max, []int{4, 8})
}

func TestParseRenderArgsAcceptsWidths(t *testing.T) {
	context, err := parseRenderArgs([]string{"--widths", "1920,1080"})
	if err != nil {
		t.Fatalf("parse widths: %v", err)
	}
	assertIntSlice(t, context.FilmShapeOverride, []int{1920, 1080})
}

func TestParseRenderArgsRejectsLegacyWidthFlag(t *testing.T) {
	if _, err := parseRenderArgs([]string{"--width", "1920"}); err == nil {
		t.Fatal("expected Engine to reject Studio-only width compatibility flag")
	}
}

func TestParseRenderArgsRejectsInvalidPixelWindow(t *testing.T) {
	_, err := parseRenderArgs([]string{"--pixel-window", "10:10,0:1"})
	if err == nil {
		t.Fatal("expected invalid pixel window to fail")
	}
}

func parseRenderArgs(args []string) (RenderContext, error) {
	h := NewHandler().ParseRenderArgs(args)
	return h.Context, h.err
}

func TestResolveRenderContextsExpandsRenderJobs(t *testing.T) {
	contexts := ResolveRenderContexts(&parser.Script{
		Render: parser.RenderScript{
			Samples:  8,
			CameraID: "front",
		},
		Renders: []parser.RenderScript{
			{},
			{Samples: 32, CameraID: "detail"},
		},
		Cameras: []parser.CameraScript{
			{ID: "front", Film: &camera.Film{Shape: []int{320, 200}, OutputFilm: "front.bin"}},
			{ID: "detail", Film: &camera.Film{Shape: []int{640, 400}, OutputFilm: "detail.bin"}},
		},
	}, RenderContext{CameraIndex: -1})

	if len(contexts) != 2 {
		t.Fatalf("expected two render contexts, got %d", len(contexts))
	}
	if contexts[0].Samples != 8 || contexts[0].CameraIndex != 0 || contexts[0].OutputFilm != "front.bin" {
		t.Fatalf("unexpected first render context: %+v", contexts[0])
	}
	if contexts[1].Samples != 32 || contexts[1].CameraIndex != 1 || contexts[1].OutputFilm != "detail.bin" {
		t.Fatalf("unexpected second render context: %+v", contexts[1])
	}
}

func TestResolveRenderContextPrefersRequestedPixelWindows(t *testing.T) {
	context := ResolveRenderContext(&parser.Script{
		Render: parser.RenderScript{CameraID: "main"},
		Cameras: []parser.CameraScript{{ID: "main", Film: &camera.Film{
			Shape: []int{10, 10},
			PixelWindows: []camera.PixelWindow{
				{Min: []int{1, 1}, Max: []int{2, 2}},
			},
		}}},
	}, RenderContext{
		CameraIndex: -1,
		PixelWindows: []camera.PixelWindow{
			{Min: []int{3, 3}, Max: []int{4, 4}},
		},
	})

	if len(context.PixelWindows) != 1 {
		t.Fatalf("expected one pixel window, got %d", len(context.PixelWindows))
	}
	assertIntSlice(t, context.PixelWindows[0].Min, []int{3, 3})
	assertIntSlice(t, context.PixelWindows[0].Max, []int{4, 4})
}

func TestResolveRenderContextsRenderJobInheritsCameraIDWhenOmitted(t *testing.T) {
	contexts := ResolveRenderContexts(&parser.Script{
		Render: parser.RenderScript{CameraID: "inherited"},
		Renders: []parser.RenderScript{
			{},
			{CameraID: "requested"},
		},
		Cameras: []parser.CameraScript{
			{ID: "requested", Film: &camera.Film{Shape: []int{10, 10}}},
			{ID: "unused", Film: &camera.Film{Shape: []int{10, 10}}},
			{ID: "inherited", Film: &camera.Film{Shape: []int{10, 10}}},
		},
	}, RenderContext{CameraIndex: -1})

	if contexts[0].CameraIndex != 2 {
		t.Fatalf("expected first render job to inherit camera index 2, got %d", contexts[0].CameraIndex)
	}
	if contexts[1].CameraIndex != 0 {
		t.Fatalf("expected second render job to use requested camera index 0, got %d", contexts[1].CameraIndex)
	}
}

func TestSelectRenderCameraPreparesHyperbolicCamera(t *testing.T) {
	cam := &camera.HyperbolicCamera{Camera3D: camera.Camera3D{
		Position:     mat.NewVecDense(3, []float64{0, 0, 0}),
		Coordinates:  []*mat.VecDense{mat.NewVecDense(3, []float64{1, 0, 0}), mat.NewVecDense(3, []float64{0, -1, 0}), mat.NewVecDense(3, []float64{0, 0, 1})},
		FieldOfViews: []float64{70, 70},
	}}
	h := NewHandler()
	h.Scene.Cameras = []camera.RayCamera{cam}

	selected, err := h.selectRenderCamera(0)
	if err != nil {
		t.Fatalf("select camera: %v", err)
	}
	if selected != cam {
		t.Fatal("expected selected hyperbolic camera")
	}
}

func TestSelectRenderCameraRequiresCamera(t *testing.T) {
	h := NewHandler()
	_, err := h.selectRenderCamera(0)
	if err == nil {
		t.Fatal("expected selecting without cameras to fail")
	}
}

func TestSelectRenderCameraPreparesSphericalCamera(t *testing.T) {
	cam := &camera.SphericalCamera{
		Camera:       camera.Camera{Film: camera.NewFilm(10, 10)},
		Position:     mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		Coordinates:  []*mat.VecDense{mat.NewVecDense(4, []float64{0, 1, 0, 0}), mat.NewVecDense(4, []float64{0, 0, 0, 1}), mat.NewVecDense(4, []float64{0, 0, 1, 0})},
		FieldOfViews: []float64{70, 70},
	}
	h := NewHandler()
	h.Scene.Cameras = []camera.RayCamera{cam}

	selected, err := h.selectRenderCamera(0)
	if err != nil {
		t.Fatalf("select camera: %v", err)
	}
	if selected != cam {
		t.Fatal("expected selected spherical camera")
	}
}

func TestConfigureRenderContextRejectsOutOfBoundsPixelWindow(t *testing.T) {
	cam := &camera.SphericalCamera{
		Position:     mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		Coordinates:  []*mat.VecDense{mat.NewVecDense(4, []float64{0, 1, 0, 0}), mat.NewVecDense(4, []float64{0, 0, 0, 1}), mat.NewVecDense(4, []float64{0, 0, 1, 0})},
		FieldOfViews: []float64{70, 70},
	}
	h := NewHandler()
	h.Scene.Cameras = []camera.RayCamera{cam}

	h.ConfigureRenderContext(RenderContext{
		CameraIndex:       0,
		FilmShapeOverride: []int{10, 10},
		PixelWindows:      []camera.PixelWindow{{Min: []int{9, 0}, Max: []int{11, 1}}},
	})

	if h.err == nil {
		t.Fatal("expected out-of-bounds pixel window to fail")
	}
}

func TestConfigureRenderContextRejectsBDPTInCurvedGeometry(t *testing.T) {
	h := NewHandler()
	h.Scene.Geometry = geometry.Klein()
	h.ConfigureRenderContext(RenderContext{
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
