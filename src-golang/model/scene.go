package model

import "src-golang/model/object"

type Scene struct {
	ObjectTree *object.ObjectTree `json:"object_tree"`
	Cameras    []*Camera          `json:"cameras"`
}

func NewScene() *Scene {
	return &Scene{
		ObjectTree: &object.ObjectTree{},
	}
}
