package medium

import (
	"fmt"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"github.com/Algo2147483647/ray/engine/model/optics/spectrum_parameter"
)

type Medium interface {
	ID() MediumID
	Name() string
	IOR(ctx bxdf.ShadingContext) float64
	SigmaA(ctx bxdf.ShadingContext) core.Spectrum
	SigmaS(ctx bxdf.ShadingContext) core.Spectrum
	IsVacuum() bool
}

type MediumID uint32

const (
	MediumNone MediumID = 0
	MediumAir  MediumID = 1
)

type Homogeneous struct {
	id     MediumID
	name   string
	eta    Model
	sigmaA optics.SpectralParameter
	sigmaS optics.SpectralParameter
}

func NewHomogeneous(id MediumID, name string, eta Model, sigmaA, sigmaS optics.SpectralParameter) Homogeneous {
	if eta == nil {
		eta = NewConstant(1)
	}
	if sigmaA == nil {
		sigmaA = spectrum_parameter.NewConstantParameter(0)
	}
	if sigmaS == nil {
		sigmaS = spectrum_parameter.NewConstantParameter(0)
	}
	return Homogeneous{
		id:     id,
		name:   name,
		eta:    eta,
		sigmaA: sigmaA,
		sigmaS: sigmaS,
	}
}

func (h Homogeneous) ID() MediumID {
	return h.id
}

func (h Homogeneous) Name() string {
	return h.name
}

func (h Homogeneous) IOR(ctx core.ShadingContext) float64 {
	wavelength := ctx.WavelengthNM
	if wavelength <= 0 && len(ctx.WavelengthsNM) > 0 {
		wavelength = ctx.WavelengthsNM[0]
	}
	eta := h.eta.Evaluate(wavelength)
	if !ior.IsValidEta(eta) {
		return 1
	}
	return eta
}

func (h Homogeneous) SigmaA(ctx core.ShadingContext) optics.Spectrum {
	return h.sigmaA.Eval(ctx)
}

func (h Homogeneous) SigmaS(ctx core.ShadingContext) optics.Spectrum {
	return h.sigmaS.Eval(ctx)
}

func (h Homogeneous) IsVacuum() bool {
	return h.sigmaA.Bounds().Max.MaxComponent() == 0 && h.sigmaS.Bounds().Max.MaxComponent() == 0
}

type Registry struct {
	mediaByID map[MediumID]Medium
	idByName  map[string]MediumID
	nextID    MediumID
}

func NewRegistry() *Registry {
	r := &Registry{
		mediaByID: make(map[MediumID]Medium),
		idByName:  make(map[string]MediumID),
		nextID:    core.MediumAir + 1,
	}
	r.Set(core.MediumAir, "air", NewHomogeneous(core.MediumAir, "air", ior.NewConstant(1), nil, nil))
	return r
}

func (r *Registry) Set(id MediumID, name string, m Medium) {
	if r == nil || id == core.MediumNone || name == "" || m == nil {
		return
	}
	r.mediaByID[id] = m
	r.idByName[name] = id
	if id >= r.nextID {
		r.nextID = id + 1
	}
}

func (r *Registry) RegisterHomogeneous(name string, eta ior.Model, sigmaA, sigmaS core.SpectralParameter) (MediumID, error) {
	if r == nil {
		return core.MediumNone, fmt.Errorf("medium registry is nil")
	}
	if name == "" {
		return core.MediumNone, fmt.Errorf("medium name must not be empty")
	}
	if existing, ok := r.idByName[name]; ok {
		r.Set(existing, name, NewHomogeneous(existing, name, eta, sigmaA, sigmaS))
		return existing, nil
	}
	id := r.nextID
	r.Set(id, name, NewHomogeneous(id, name, eta, sigmaA, sigmaS))
	return id, nil
}

func (r *Registry) ID(name string) (MediumID, bool) {
	if r == nil {
		return core.MediumNone, false
	}
	id, ok := r.idByName[name]
	return id, ok
}

func (r *Registry) Get(id MediumID) Medium {
	if r == nil {
		return nil
	}
	return r.mediaByID[id]
}

func (r *Registry) IOR(id MediumID, ctx core.ShadingContext) float64 {
	if id == core.MediumNone {
		id = core.MediumAir
	}
	if r == nil {
		return 1
	}
	m := r.Get(id)
	if m == nil {
		return 1
	}
	return m.IOR(ctx)
}
