package medium

import (
	"fmt"
)

type Medium interface {
	ID() MediumID
	Name() string
	IOR(ctx WavelengthContext) float64
}

type WavelengthContext interface {
	SpectralWavelengthNM() float64
	SpectralWavelengthsNM() []float64
}

type MediumID uint32

const (
	MediumNone MediumID = 0
	MediumAir  MediumID = 1
)

type Homogeneous struct {
	id   MediumID
	name string
	eta  Model
}

func NewHomogeneous(id MediumID, name string, eta Model) Homogeneous {
	if eta == nil {
		eta = NewConstant(1)
	}
	return Homogeneous{
		id:   id,
		name: name,
		eta:  eta,
	}
}

func (h Homogeneous) ID() MediumID {
	return h.id
}

func (h Homogeneous) Name() string {
	return h.name
}

func (h Homogeneous) IOR(ctx WavelengthContext) float64 {
	wavelength := 0.0
	if ctx != nil {
		wavelength = ctx.SpectralWavelengthNM()
		if wavelength <= 0 {
			wavelengths := ctx.SpectralWavelengthsNM()
			if len(wavelengths) > 0 {
				wavelength = wavelengths[0]
			}
		}
	}
	eta := h.eta.Evaluate(wavelength)
	if !IsValidEta(eta) {
		return 1
	}
	return eta
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
		nextID:    MediumAir + 1,
	}
	r.Set(MediumAir, "air", NewHomogeneous(MediumAir, "air", NewConstant(1)))
	return r
}

func (r *Registry) Set(id MediumID, name string, m Medium) {
	if r == nil || id == MediumNone || name == "" || m == nil {
		return
	}
	r.mediaByID[id] = m
	r.idByName[name] = id
	if id >= r.nextID {
		r.nextID = id + 1
	}
}

func (r *Registry) RegisterHomogeneous(name string, eta Model) (MediumID, error) {
	if r == nil {
		return MediumNone, fmt.Errorf("medium registry is nil")
	}
	if name == "" {
		return MediumNone, fmt.Errorf("medium name must not be empty")
	}
	if existing, ok := r.idByName[name]; ok {
		r.Set(existing, name, NewHomogeneous(existing, name, eta))
		return existing, nil
	}
	id := r.nextID
	r.Set(id, name, NewHomogeneous(id, name, eta))
	return id, nil
}

func (r *Registry) ID(name string) (MediumID, bool) {
	if r == nil {
		return MediumNone, false
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

func (r *Registry) IOR(id MediumID, ctx WavelengthContext) float64 {
	if id == MediumNone {
		id = MediumAir
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
