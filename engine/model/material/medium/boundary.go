package medium

type Boundary struct {
	Outside  MediumID
	Inside   MediumID
	Priority int
	Thin     bool
}

func NewBoundary(outside, inside MediumID) Boundary {
	if outside == MediumNone {
		outside = MediumAir
	}
	return Boundary{
		Outside: outside,
		Inside:  inside,
	}
}

func (b Boundary) Active() bool {
	return b.Inside != MediumNone
}
