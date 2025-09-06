package model

import (
	"src-golang/model/camera"
	"src-golang/model/object"
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
