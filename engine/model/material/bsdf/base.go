package bsdf

import "github.com/Algo2147483647/ray/engine/model/material/core"

type BSDF interface {
	core.Scattering
}
