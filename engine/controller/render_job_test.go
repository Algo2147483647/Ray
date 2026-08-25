package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/film"
)

func testRenderSpec() parser.RenderScript {
	return parser.RenderScript{CameraID: "main", Film: &film.Spec{Shape: []int{4, 4}}, Output: "test.bin"}
}

func testRenderScene() *model.Scene {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	scene.Cameras["main"] = camera.NewCamera3D()
	return scene
}

func TestResolveRenderJobOwnsRenderDefaults(t *testing.T) {
	job, err := ResolveRenderJob(testRenderSpec(), testRenderScene())
	if err != nil {
		t.Fatal(err)
	}
	if job.Integrator() != "path" || job.Samples() != 20 || job.ThreadNum() <= 0 {
		t.Fatalf("unexpected resolved defaults: integrator=%q samples=%d threads=%d", job.Integrator(), job.Samples(), job.ThreadNum())
	}
	if job.WavelengthSamples() != 1 {
		t.Fatalf("wavelength sample default = %d, want 1", job.WavelengthSamples())
	}
	if job.Film().SpectralBinCount() != film.DefaultSpectralBinCount {
		t.Fatalf("spectral bins = %d, want %d", job.Film().SpectralBinCount(), film.DefaultSpectralBinCount)
	}
}

func TestResolveRenderJobPreservesExplicitCounts(t *testing.T) {
	spec := testRenderSpec()
	samples, threads, wavelengths := int64(7), 2, 4
	spec.Samples = &samples
	spec.ThreadNum = &threads
	spec.WavelengthSamples = &wavelengths
	job, err := ResolveRenderJob(spec, testRenderScene())
	if err != nil {
		t.Fatal(err)
	}
	if job.Samples() != samples || job.ThreadNum() != threads || job.WavelengthSamples() != wavelengths {
		t.Fatalf("explicit counts changed: samples=%d threads=%d wavelengths=%d", job.Samples(), job.ThreadNum(), job.WavelengthSamples())
	}
}
