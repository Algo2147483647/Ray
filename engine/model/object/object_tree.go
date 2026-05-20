package object

import "github.com/Algo2147483647/ray/engine/model/material/medium"

type BVHUpdateStrategy string

const (
	BVHUpdateAuto    BVHUpdateStrategy = "auto"
	BVHUpdateRefit   BVHUpdateStrategy = "refit"
	BVHUpdateRebuild BVHUpdateStrategy = "rebuild"
)

type ObjectTree struct {
	Root        *ObjectNode
	Objects     []*Object
	ObjectNodes []*ObjectNode
	Media       *medium.Registry
}

func (t *ObjectTree) AddObject(object *Object) *Object {
	t.Objects = append(t.Objects, object)
	return t.Objects[len(t.Objects)-1]
}
