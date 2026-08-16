package ray_tracing

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

// IntegratorKind is the serialized name of a light-transport algorithm.
type IntegratorKind string

const (
	IntegratorPathTracing  IntegratorKind = "path"
	IntegratorBDPT         IntegratorKind = "bdpt"
	IntegratorLightTracing IntegratorKind = "light_tracing"
)

// ParseIntegratorKind accepts canonical names and compatibility aliases at the
func ParseIntegratorKind(value string) (IntegratorKind, error) {
	switch value {
	case "", string(IntegratorPathTracing):
		return IntegratorPathTracing, nil
	case string(IntegratorBDPT):
		return IntegratorBDPT, nil
	case string(IntegratorLightTracing), "light_trace":
		return IntegratorLightTracing, nil
	default:
		return "", fmt.Errorf("unsupported integrator %q", value)
	}
}

// RenderContext contains the scene-level inputs shared by all integrators.
type RenderContext struct {
	Camera     camera.RayCamera
	ObjectTree *object.ObjectTree
	Samples    int64
}

// SceneIntegrator owns the complete lifecycle of one configured render.
type SceneIntegrator interface {
	Render(RenderContext) error
}

// RenderDriver owns a work-distribution model. Algorithms are injected as
// kernels, so pixel-driven and splat-driven transports share one lifecycle
// without pretending to have the same scheduling semantics.
type RenderDriver interface {
	Run(*RenderSession) error
	EffectiveSampleCount(*RenderSession) int64
	ConcurrentFilmWrites() bool
}

type configuredSceneIntegrator struct {
	handler *Handler
	driver  RenderDriver
}

// NewSceneIntegrator resolves configuration once, before rendering begins.
func NewSceneIntegrator(kind IntegratorKind, handler *Handler) (SceneIntegrator, error) {
	if handler == nil {
		return nil, fmt.Errorf("integrator handler is nil")
	}

	switch kind {
	case IntegratorPathTracing:
		return &configuredSceneIntegrator{
			handler: handler,
			driver:  &pixelDriver{kernel: pathTracingKernel{}},
		}, nil

	case IntegratorBDPT:
		return &configuredSceneIntegrator{
			handler: handler,
			driver:  &splatDriver{kernel: &bdptKernel{}},
		}, nil

	case IntegratorLightTracing:
		return &configuredSceneIntegrator{
			handler: handler,
			driver:  &splatDriver{kernel: &lightTracingKernel{}},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported integrator %q", kind)
	}
}

func (i *configuredSceneIntegrator) Render(ctx RenderContext) error {
	session, err := newRenderSession(i.handler, ctx, i.driver.ConcurrentFilmWrites())
	if err != nil {
		return err
	}

	err = i.driver.Run(session)
	if err != nil {
		return err
	}

	session.Context.Camera.GetFilm().Samples = i.driver.EffectiveSampleCount(session)
	return nil
}
