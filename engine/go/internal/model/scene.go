package model

import (
	"github.com/Algo2147483647/ray/engine/go/internal/model/camera"
	"github.com/Algo2147483647/ray/engine/go/internal/model/object"
)

type Scene struct {
	ObjectTree *object.ObjectTree `json:"object_tree"`
	Cameras    []camera.Camera    `json:"cameras"`
}

func NewScene() *Scene {
	return &Scene{
		ObjectTree: &object.ObjectTree{},
	}
}
