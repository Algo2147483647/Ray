package model

import (
	"src-golang/model/object"
	"src-golang/model/optics"
)

type Scene struct {
	ObjectTree *object.ObjectTree `json:"object_tree"`
	Cameras    []*optics.Camera   `json:"cameras"`
}

func NewScene() *Scene {
	return &Scene{
		ObjectTree: &object.ObjectTree{},
	}
}
