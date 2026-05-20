package object

import (
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/shape"
)

type Object struct {
	Shape          shape.Shape
	Material       *material.Material
	MediumBoundary medium.Boundary
}
