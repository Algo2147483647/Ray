package ray_tracing

import (
	"fmt"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
	"gonum.org/v1/gonum/mat"
)

// lightTracingKernel implements the t=1 algorithm while splatSceneIntegrator owns
// scheduling, synchronization, normalization and progress reporting.
type lightTracingKernel struct{}

type lightTracingPreparedState struct {
	projective  camera.ProjectiveCamera
	lights      []areaLight
	totalWeight float64
	activeMask  []bool
	pixelCount  int
	width       int
	height      int
	totalPaths  int64
}

func (*lightTracingPreparedState) preparedIntegratorState() {}

func (k *lightTracingKernel) Prepare(context *RenderContext) (PreparedIntegratorState, error) {
	projective, ok := context.Target.Camera.(camera.ProjectiveCamera)
	if !ok {
		return nil, fmt.Errorf("light tracing requires a projective camera, got %T", context.Target.Camera)
	}
	state := &lightTracingPreparedState{projective: projective}
	state.lights, state.totalWeight = collectAreaLights(context.ObjectTree)
	film := context.Target.Film
	if len(film.Shape) != 2 {
		return nil, fmt.Errorf("light tracing requires a 2D Film")
	}
	state.width = film.Shape[0]
	state.height = film.Shape[1]
	state.pixelCount = film.ElementCount()
	state.activeMask = make([]bool, state.pixelCount)
	activePixels := int64(state.pixelCount)
	if len(film.PixelWindows) == 0 {
		for pixel := range state.activeMask {
			state.activeMask[pixel] = true
		}
	} else {
		state.activeMask, activePixels = buildPixelWindowMask(
			film.Shape,
			film.PixelWindows,
		)
	}
	if len(state.lights) == 0 || state.totalWeight <= 0 || activePixels <= 0 {
		return state, nil
	}
	state.totalPaths = context.Samples * activePixels
	return state, nil
}

func (k *lightTracingKernel) WorkCount(_ *RenderContext, prepared PreparedIntegratorState) int64 {
	state, ok := prepared.(*lightTracingPreparedState)
	if !ok || state == nil {
		return 0
	}
	return state.totalPaths
}

func (k *lightTracingKernel) TraceSample(context *RenderContext, prepared PreparedIntegratorState, _ int64) []FilmSplat {
	state, ok := prepared.(*lightTracingPreparedState)
	if !ok || state == nil {
		return nil
	}
	wavelength := context.Handler.wavelengthSampler().Sample(rand.Float64())
	wavelengthNM, wavelengthPDF := wavelength.LambdaNM, wavelength.PDF
	path := context.Handler.buildLightSubpath(
		context.ObjectTree,
		state.lights,
		state.totalWeight,
		wavelengthNM,
		wavelengthPDF,
	)
	splats := make([]FilmSplat, 0, len(path))
	for vertexIndex := range path {
		value, projection, valid := projectLightVertex(
			state.projective,
			context.Target.Film.Shape,
			context.ObjectTree,
			&path[vertexIndex],
		)
		if !valid {
			continue
		}
		pixel, ok := camera.PixelIndex(projection.Position[0], projection.Position[1], state.width, state.height)
		if !ok || !state.activeMask[pixel] {
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
	filmShape []int,
	tree *object.ObjectTree,
	vertex *bdptVertex,
) (float64, camera.FilmProjection, bool) {
	if vertex == nil || vertex.Point == nil || vertex.GeometricNormal == nil || vertex.Object == nil || vertex.Object.Material == nil {
		return 0, camera.FilmProjection{}, false
	}
	projection, ok := renderCamera.ProjectPoint(vertex.Point, filmShape)
	if !ok {
		return 0, camera.FilmProjection{}, false
	}
	cameraPoint := mat.VecDenseCopyOf(vertex.Point)
	cameraPoint.AddScaledVec(cameraPoint, projection.Distance, projection.ToCamera)
	if !visibleSegment(tree, vertex.Point, cameraPoint, projection.ToCamera, projection.Distance) {
		return 0, camera.FilmProjection{}, false
	}

	cosCamera := absDot(vertex.GeometricNormal, projection.ToCamera)
	if cosCamera <= 0 {
		return 0, camera.FilmProjection{}, false
	}
	factor := projection.Jacobian * cosCamera
	transmittance := evaluateSegmentTransmittance(
		getMediumRegistry(tree),
		bdptSegmentMedium(vertex, projection.ToCamera),
		projection.Distance,
		vertex.Context,
	)

	if vertex.Kind == bdptVertexLight {
		wo := vertex.emissionLocal(projection.ToCamera)
		emitted, ok := powerAtWavelength(vertex.Object.Material.Emission.Eval(vertex.Context, wo), vertex.Context.WavelengthNM)
		if !ok {
			return 0, camera.FilmProjection{}, false
		}
		value := transmittance.ApplyToPower(emitted * vertex.Beta * factor)
		return value, projection, validPower(value)
	}
	if !vertex.Object.Material.HasSurface() || !vertex.Connectible {
		return 0, camera.FilmProjection{}, false
	}

	wi := vertex.Frame.WorldToLocal(projection.ToCamera)
	f := vertex.Object.Material.Surface.Eval(vertex.Context, wi, vertex.WoLocal)
	if f.IsZero() {
		return 0, camera.FilmProjection{}, false
	}
	fPower, ok := powerAtWavelength(f, vertex.Context.WavelengthNM)
	if !ok {
		return 0, camera.FilmProjection{}, false
	}
	value := transmittance.ApplyToPower(vertex.Beta * fPower * factor)
	return value, projection, validPower(value)
}
