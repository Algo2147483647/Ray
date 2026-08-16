package ray_tracing

import (
	"strings"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

func TestParseIntegratorKindCanonicalizesLightTraceAlias(t *testing.T) {
	for _, input := range []string{"light_tracing", "light_trace"} {
		kind, err := ParseIntegratorKind(input)
		if err != nil {
			t.Fatalf("ParseIntegratorKind(%q): %v", input, err)
		}
		if kind != IntegratorLightTracing {
			t.Fatalf("ParseIntegratorKind(%q) = %q, want %q", input, kind, IntegratorLightTracing)
		}
	}
}

func TestNewSceneIntegratorSelectsRuntimeImplementation(t *testing.T) {
	handler := NewHandler()
	tests := []struct {
		kind       IntegratorKind
		wantDriver string
		wantKernel string
	}{
		{IntegratorPathTracing, "pixel", "path"},
		{IntegratorBDPT, "splat", "bdpt"},
		{IntegratorLightTracing, "splat", "light"},
	}
	for _, test := range tests {
		integrator, err := NewSceneIntegrator(test.kind, handler)
		if err != nil {
			t.Fatalf("NewSceneIntegrator(%q): %v", test.kind, err)
		}
		configured, ok := integrator.(*configuredSceneIntegrator)
		if !ok {
			t.Fatalf("kind %q selected %T", test.kind, integrator)
		}
		switch driver := configured.driver.(type) {
		case *pixelDriver:
			if test.wantDriver != "pixel" {
				t.Fatalf("kind %q selected pixel driver, want %s", test.kind, test.wantDriver)
			}
			switch driver.kernel.(type) {
			case pathTracingKernel:
				if test.wantKernel != "path" {
					t.Fatalf("kind %q selected path kernel, want %s", test.kind, test.wantKernel)
				}
			default:
				t.Fatalf("kind %q selected unexpected pixel kernel %T", test.kind, driver.kernel)
			}
		case *splatDriver:
			if test.wantDriver != "splat" {
				t.Fatalf("kind %q selected splat driver, want %s", test.kind, test.wantDriver)
			}
			switch driver.kernel.(type) {
			case *lightTracingKernel:
				if test.wantKernel != "light" {
					t.Fatalf("kind %q selected light kernel, want %s", test.kind, test.wantKernel)
				}
			case *bdptKernel:
				if test.wantKernel != "bdpt" {
					t.Fatalf("kind %q selected bdpt kernel, want %s", test.kind, test.wantKernel)
				}
			default:
				t.Fatalf("kind %q selected unexpected splat kernel %T", test.kind, driver.kernel)
			}
		default:
			t.Fatalf("kind %q selected unexpected driver %T", test.kind, configured.driver)
		}
	}
}

func TestLightTracingRejectsNonProjectiveCamera(t *testing.T) {
	integrator, err := NewSceneIntegrator(IntegratorLightTracing, NewHandler())
	if err != nil {
		t.Fatalf("NewSceneIntegrator: %v", err)
	}
	err = integrator.Render(RenderContext{
		Camera:     fixedCamera{Camera: camera.Camera{Film: camera.NewFilm(1, 1)}},
		ObjectTree: &object.ObjectTree{},
		Samples:    1,
	})
	if err == nil {
		t.Fatal("light tracing accepted a non-projective camera")
	}
	if !strings.Contains(err.Error(), "projective camera") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBDPTWorkCountIncludesSampledWavelengthStrata(t *testing.T) {
	handler := NewHandler()
	handler.SceneGeometry = geometry.Euclidean()
	handler.SpectrumMode = optics.SpectrumModeSampledWavelengths
	handler.WavelengthSamples = 4
	session, err := newRenderSession(handler, RenderContext{
		Camera: fixedCamera{Camera: camera.Camera{Film: camera.NewFilm(2, 1)}}, ObjectTree: (&object.ObjectTree{}).Build(),
		Samples: 3,
	}, true)
	if err != nil {
		t.Fatalf("newRenderSession: %v", err)
	}
	kernel := &bdptKernel{}
	if err := kernel.Prepare(session); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if got, want := kernel.WorkCount(session), int64(2*3*4); got != want {
		t.Fatalf("BDPT work count = %d, want %d", got, want)
	}
}

func TestTraceSceneRejectsUnknownIntegrator(t *testing.T) {
	handler := NewHandler()
	handler.IntegratorKind = IntegratorKind("unknown")
	err := handler.TraceScene(fixedCamera{Camera: camera.Camera{Film: camera.NewFilm(1, 1)}}, nil, 1)
	if err == nil {
		t.Fatal("TraceScene accepted an unknown integrator")
	}
}
