package medium

import "github.com/Algo2147483647/ray/engine/model/material/core"

type Boundary struct {
	Outside  core.MediumID
	Inside   core.MediumID
	Priority int
	Thin     bool
}

func NewBoundary(outside, inside core.MediumID) Boundary {
	if outside == core.MediumNone {
		outside = core.MediumAir
	}
	return Boundary{
		Outside: outside,
		Inside:  inside,
	}
}

func (b Boundary) Active() bool {
	return b.Inside != core.MediumNone
}
