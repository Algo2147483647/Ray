package emission

import "github.com/Algo2147483647/ray/engine/go/internal/material/core"

type Constant struct {
	Color core.Spectrum
}

func NewConstant(color core.Spectrum) Constant {
	return Constant{Color: color}
}

func (e Constant) Emit(core.ShadingContext, core.Direction) core.Spectrum {
	return e.Color
}

func (e Constant) IsDelta() bool {
	return false
}
