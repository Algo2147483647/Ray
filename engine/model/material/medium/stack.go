package medium

import "github.com/Algo2147483647/ray/engine/model/material/core"

const baseMediumPriority = -1 << 30

type StackEntry struct {
	ID       core.MediumID
	Priority int
}

type Transition struct {
	Incident core.MediumID
	Transmit core.MediumID
	Entering bool
	Thin     bool
}

type Stack struct {
	entries []StackEntry
}

func NewStack(initial core.MediumID) Stack {
	var s Stack
	s.Reset(initial)
	return s
}

func (s *Stack) Reset(initial core.MediumID) {
	s.entries = s.entries[:0]
	if initial == core.MediumNone {
		initial = core.MediumAir
	}
	s.entries = append(s.entries, StackEntry{ID: initial, Priority: baseMediumPriority})
}

func (s Stack) Current() core.MediumID {
	if len(s.entries) == 0 {
		return core.MediumAir
	}
	best := s.entries[0]
	for _, entry := range s.entries[1:] {
		if entry.Priority >= best.Priority {
			best = entry
		}
	}
	return best.ID
}

func (s *Stack) Push(id core.MediumID) {
	s.PushWithPriority(id, 0)
}

func (s *Stack) PushWithPriority(id core.MediumID, priority int) {
	if id == core.MediumNone {
		return
	}
	s.entries = append(s.entries, StackEntry{ID: id, Priority: priority})
}

func (s *Stack) Remove(id core.MediumID) bool {
	if id == core.MediumNone {
		return false
	}
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].ID == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			if len(s.entries) == 0 {
				s.entries = append(s.entries, StackEntry{ID: core.MediumAir, Priority: baseMediumPriority})
			}
			return true
		}
	}
	return false
}

func (s *Stack) RemoveWithPriority(id core.MediumID, priority int) bool {
	if id == core.MediumNone {
		return false
	}
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i].ID == id && s.entries[i].Priority == priority {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			if len(s.entries) == 0 {
				s.entries = append(s.entries, StackEntry{ID: core.MediumAir, Priority: baseMediumPriority})
			}
			return true
		}
	}
	return s.Remove(id)
}

func (s *Stack) EnterBoundary(boundary Boundary) {
	if !boundary.Active() || boundary.Thin {
		return
	}
	s.PushWithPriority(boundary.Inside, boundary.Priority)
}

func (s *Stack) ExitBoundary(boundary Boundary) {
	if !boundary.Active() || boundary.Thin {
		return
	}
	s.RemoveWithPriority(boundary.Inside, boundary.Priority)
}

func (s Stack) ResolveTransition(boundary Boundary, entering bool) Transition {
	incident := s.Current()
	transmit := incident
	if !boundary.Active() {
		return Transition{Incident: incident, Transmit: transmit, Entering: false}
	}

	if boundary.Thin {
		if entering {
			transmit = boundary.Inside
		} else if boundary.Outside != core.MediumNone {
			transmit = boundary.Outside
		}
		return Transition{
			Incident: incident,
			Transmit: transmit,
			Entering: entering,
			Thin:     true,
		}
	}

	candidate := s.Clone()
	if entering {
		candidate.EnterBoundary(boundary)
	} else {
		candidate.ExitBoundary(boundary)
	}
	transmit = candidate.Current()
	return Transition{
		Incident: incident,
		Transmit: transmit,
		Entering: entering,
	}
}

func (s Stack) Contains(id core.MediumID) bool {
	for _, entry := range s.entries {
		if entry.ID == id {
			return true
		}
	}
	return false
}

func (s Stack) Clone() Stack {
	entries := make([]StackEntry, len(s.entries))
	copy(entries, s.entries)
	return Stack{entries: entries}
}

func (s Stack) Entries() []core.MediumID {
	ids := make([]core.MediumID, len(s.entries))
	for i, entry := range s.entries {
		ids[i] = entry.ID
	}
	return ids
}

func (s Stack) DetailedEntries() []StackEntry {
	entries := make([]StackEntry, len(s.entries))
	copy(entries, s.entries)
	return entries
}
