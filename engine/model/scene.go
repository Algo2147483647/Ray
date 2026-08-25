package model

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

type Scene struct {
	ObjectTree *object.ObjectTree          `json:"object_tree"`
	Cameras    map[string]camera.RayCamera `json:"cameras"`
	Space      geometry.SceneSpace         `json:"-"`
	MaxArc     float64                     `json:"-"` // 0 ⇒ unbounded
}

func NewScene(space geometry.SceneSpace) *Scene {
	return &Scene{
		ObjectTree: &object.ObjectTree{},
		Cameras:    make(map[string]camera.RayCamera),
		Space:      geometry.NewSceneSpace(space.Geometry, space.Dimension),
	}
}
