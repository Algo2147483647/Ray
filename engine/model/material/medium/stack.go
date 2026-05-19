package medium

import "github.com/Algo2147483647/ray/engine/material/core"

type Stack struct {
	entries []core.MediumID
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
	s.entries = append(s.entries, initial)
}

func (s Stack) Current() core.MediumID {
	if len(s.entries) == 0 {
		return core.MediumAir
	}
	return s.entries[len(s.entries)-1]
}

func (s *Stack) Push(id core.MediumID) {
	if id == core.MediumNone {
		return
	}
	s.entries = append(s.entries, id)
}

func (s *Stack) Remove(id core.MediumID) bool {
	if id == core.MediumNone {
		return false
	}
	for i := len(s.entries) - 1; i >= 0; i-- {
		if s.entries[i] == id {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			if len(s.entries) == 0 {
				s.entries = append(s.entries, core.MediumAir)
			}
			return true
		}
	}
	return false
}

func (s Stack) Contains(id core.MediumID) bool {
	for _, entry := range s.entries {
		if entry == id {
			return true
		}
	}
	return false
}

func (s Stack) Clone() Stack {
	entries := make([]core.MediumID, len(s.entries))
	copy(entries, s.entries)
	return Stack{entries: entries}
}

func (s Stack) Entries() []core.MediumID {
	entries := make([]core.MediumID, len(s.entries))
	copy(entries, s.entries)
	return entries
}
