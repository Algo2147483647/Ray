package medium

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/model/material/core"
)

func TestStackTracksNestedMedia(t *testing.T) {
	stack := NewStack(core.MediumAir)
	glass := core.MediumID(2)
	water := core.MediumID(3)

	stack.Push(glass)
	stack.Push(water)
	if got := stack.Current(); got != water {
		t.Fatalf("unexpected current medium: got %d want %d", got, water)
	}
	if !stack.Contains(glass) || !stack.Contains(water) {
		t.Fatal("expected stack to contain nested media")
	}

	if !stack.Remove(water) {
		t.Fatal("expected water removal to succeed")
	}
	if got := stack.Current(); got != glass {
		t.Fatalf("unexpected current medium after water exit: got %d want %d", got, glass)
	}

	if !stack.Remove(glass) {
		t.Fatal("expected glass removal to succeed")
	}
	if got := stack.Current(); got != core.MediumAir {
		t.Fatalf("unexpected current medium after glass exit: got %d want air", got)
	}
}

func TestStackCloneIsIndependent(t *testing.T) {
	stack := NewStack(core.MediumAir)
	stack.Push(core.MediumID(2))

	clone := stack.Clone()
	clone.Push(core.MediumID(3))

	if got := stack.Current(); got != core.MediumID(2) {
		t.Fatalf("original stack changed after clone mutation: got %d", got)
	}
	if got := clone.Current(); got != core.MediumID(3) {
		t.Fatalf("unexpected clone current medium: got %d", got)
	}
}

func TestStackCurrentUsesHighestPriorityMedium(t *testing.T) {
	stack := NewStack(core.MediumAir)
	low := core.MediumID(2)
	high := core.MediumID(3)

	stack.PushWithPriority(high, 10)
	stack.PushWithPriority(low, 1)

	if got := stack.Current(); got != high {
		t.Fatalf("expected highest priority medium, got %d want %d", got, high)
	}
}

func TestResolveTransitionKeepsLowerPriorityBoundaryHidden(t *testing.T) {
	stack := NewStack(core.MediumAir)
	high := core.MediumID(2)
	low := core.MediumID(3)
	stack.EnterBoundary(Boundary{Inside: high, Priority: 10})

	transition := stack.ResolveTransition(Boundary{Inside: low, Priority: 1}, true)
	if transition.Incident != high || transition.Transmit != high {
		t.Fatalf("expected lower priority enter to remain in high medium, got %d -> %d", transition.Incident, transition.Transmit)
	}

	stack.EnterBoundary(Boundary{Inside: low, Priority: 1})
	stack.ExitBoundary(Boundary{Inside: high, Priority: 10})
	if got := stack.Current(); got != low {
		t.Fatalf("expected lower priority medium to become active after high exit, got %d", got)
	}
}

func TestThinBoundaryResolvesInterfaceWithoutChangingStack(t *testing.T) {
	stack := NewStack(core.MediumAir)
	glass := core.MediumID(2)
	boundary := Boundary{Outside: core.MediumAir, Inside: glass, Thin: true, Priority: 5}

	enter := stack.ResolveTransition(boundary, true)
	if !enter.Thin || enter.Incident != core.MediumAir || enter.Transmit != glass {
		t.Fatalf("unexpected thin enter transition: %+v", enter)
	}

	stack.EnterBoundary(boundary)
	if got := stack.Current(); got != core.MediumAir {
		t.Fatalf("thin boundary must not push volume medium, got %d", got)
	}
}
