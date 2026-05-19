package object

import (
	"github.com/Algo2147483647/ray/engine/go/internal/material/core"
	"github.com/Algo2147483647/ray/engine/go/internal/material/medium"
	"github.com/Algo2147483647/ray/engine/go/internal/model/shape"
)

type Object struct {
	Shape          shape.Shape
	Material       *core.Material
	MediumBoundary medium.Boundary
}
