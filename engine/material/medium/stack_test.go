package medium

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/material/core"
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
