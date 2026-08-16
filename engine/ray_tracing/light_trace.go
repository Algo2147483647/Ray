package ray_tracing

import (
	"fmt"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

// lightTracingKernel implements the t=1 algorithm while splatSceneIntegrator owns
// scheduling, synchronization, normalization and progress reporting.
type lightTracingKernel struct {
	projective  camera.ProjectiveCamera
	lights      []areaLight
	totalWeight float64
	activeMask  []bool
	pixelCount  int
	width       int
	height      int
	totalPaths  int64
}

func (k *lightTracingKernel) Prepare(context *RenderContext) error {
	projective, ok := context.Camera.(camera.ProjectiveCamera)
	if !ok {
		return fmt.Errorf("light tracing requires a projective camera, got %T", context.Camera)
	}
	k.projective = projective
	k.lights, k.totalWeight = collectAreaLights(context.ObjectTree)
	film := context.Camera.GetFilm()
	if len(film.Shape) != 2 {
		return fmt.Errorf("light tracing requires a 2D Film")
	}
	k.width = film.Shape[0]
	k.height = film.Shape[1]
	k.pixelCount = film.ElementCount()
	k.activeMask = make([]bool, k.pixelCount)
	activePixels := int64(k.pixelCount)
	if len(film.PixelWindows) == 0 {
		for pixel := range k.activeMask {
			k.activeMask[pixel] = true
		}
	} else {
		k.activeMask, activePixels = buildPixelWindowMask(
			film.Shape,
			film.PixelWindows,
		)
	}
	if len(k.lights) == 0 || k.totalWeight <= 0 || activePixels <= 0 {
		k.totalPaths = 0
		return nil
	}
	k.totalPaths = context.Samples * activePixels
	return nil
}

func (k *lightTracingKernel) WorkCount(*RenderContext) int64 {
	return k.totalPaths
}

func (k *lightTracingKernel) TraceSample(context *RenderContext, _ int64) []FilmSplat {
	wavelength := context.Handler.wavelengthSampler().Sample(rand.Float64())
	wavelengthNM, wavelengthPDF := wavelength.LambdaNM, wavelength.PDF
	path := context.Handler.buildLightSubpath(
		context.ObjectTree,
		k.lights,
		k.totalWeight,
		wavelengthNM,
		wavelengthPDF,
	)
	splats := make([]FilmSplat, 0, len(path))
	for vertexIndex := range path {
		value, projection, valid := projectLightVertex(
			k.projective,
			context.ObjectTree,
			&path[vertexIndex],
		)
		if !valid {
			continue
		}
		pixel, ok := camera.PixelIndex(projection.Position[0], projection.Position[1], k.width, k.height)
		if !ok || !k.activeMask[pixel] {
			continue
		}
		splats = append(splats, FilmSplat{
			Pixel: pixel, WavelengthNM: wavelengthNM,
			WavelengthPDF: wavelengthPDF, Value: value,
		})
	}
	return splats
}

func projectLightVertex(
	renderCamera camera.ProjectiveCamera,
	tree *object.ObjectTree,
	vertex *bdptVertex,
) (optics.Spectrum, camera.FilmProjection, bool) {
	if vertex == nil || vertex.Point == nil || vertex.GeometricNormal == nil || vertex.Object == nil || vertex.Object.Material == nil {
		return optics.Spectrum{}, camera.FilmProjection{}, false
	}
	projection, ok := renderCamera.ProjectPoint(vertex.Point)
	if !ok {
		return optics.Spectrum{}, camera.FilmProjection{}, false
	}
	cameraPoint := mat.VecDenseCopyOf(vertex.Point)
	cameraPoint.AddScaledVec(cameraPoint, projection.Distance, projection.ToCamera)
	if !visibleSegment(tree, vertex.Point, cameraPoint, projection.ToCamera, projection.Distance) {
		return optics.Spectrum{}, camera.FilmProjection{}, false
	}

	cosCamera := absDot(vertex.GeometricNormal, projection.ToCamera)
	if cosCamera <= 0 {
		return optics.Spectrum{}, camera.FilmProjection{}, false
	}
	factor := projection.Jacobian * cosCamera
	transmittance := evaluateSegmentTransmittance(
		getMediumRegistry(tree),
		vertex.MediumStack.Current(),
		projection.Distance,
		vertex.Context,
	)

	if vertex.LightEndpoint {
		wo := vertex.Frame.WorldToLocal(projection.ToCamera)
		value := transmittance.ApplyToSpectrum(
			vertex.Object.Material.Emission.Emit(vertex.Context, wo).Mul(vertex.Beta),
		).MulScalar(factor)
		return value, projection, validSpectrum(value)
	}
	if !vertex.Object.Material.HasSurface() || vertex.SampledDelta ||
		vertex.Object.Material.Surface.DeltaFlags() != bxdf.DeltaNone {
		return optics.Spectrum{}, camera.FilmProjection{}, false
	}

	wi := vertex.Frame.WorldToLocal(projection.ToCamera)
	f := vertex.Object.Material.Surface.Eval(vertex.Context, wi, vertex.WoLocal)
	if f.IsZero() {
		return optics.Spectrum{}, camera.FilmProjection{}, false
	}
	value := transmittance.ApplyToSpectrum(vertex.Beta.Mul(f)).MulScalar(factor)
	return value, projection, validSpectrum(value)
}
