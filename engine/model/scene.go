package model

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

type Scene struct {
	ObjectTree *object.ObjectTree `json:"object_tree"`
	Cameras    []camera.Camera    `json:"cameras"`
	Geometry   geometry.Geometry  `json:"-"` // nil ⇒ Euclidean
	MaxArc     float64            `json:"-"` // 0 ⇒ unbounded
}

func NewScene() *Scene {
	return &Scene{
		ObjectTree: &object.ObjectTree{},
	}
}
