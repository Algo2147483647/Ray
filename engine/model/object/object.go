package object

import (
	"github.com/Algo2147483647/ray/engine/material/core"
	"github.com/Algo2147483647/ray/engine/material/medium"
	"github.com/Algo2147483647/ray/engine/model/shape"
)

type Object struct {
	Shape          shape.Shape
	Material       *core.Material
	MediumBoundary medium.Boundary
}
