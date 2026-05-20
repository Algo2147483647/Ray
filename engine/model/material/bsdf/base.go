package bsdf

import (
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
)

type BSDF interface {
	bxdf.Scattering
}
