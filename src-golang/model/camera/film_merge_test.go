package camera

import (
	"math"
	"testing"
)

func TestFilmMergeMergesAllChannelsBySampleWeight(t *testing.T) {
	base := NewFilm(1, 1)
	base.Samples = 2
	base.Data[0].Data[0] = 0.2
	base.Data[1].Data[0] = 0.4
	base.Data[2].Data[0] = 0.6

	incoming := NewFilm(1, 1)
	incoming.Samples = 1
	incoming.Data[0].Data[0] = 0.8
	incoming.Data[1].Data[0] = 0.1
	incoming.Data[2].Data[0] = 0.3

	base.Merge(incoming)

	if got := base.Data[0].Data[0]; math.Abs(got-0.4) > 1e-9 {
		t.Fatalf("unexpected red channel merge result: got %v want 0.4", got)
	}
	if got := base.Data[1].Data[0]; math.Abs(got-0.3) > 1e-9 {
		t.Fatalf("unexpected green channel merge result: got %v want 0.3", got)
	}
	if got := base.Data[2].Data[0]; math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("unexpected blue channel merge result: got %v want 0.5", got)
	}
	if base.Samples != 3 {
		t.Fatalf("unexpected merged sample count: got %d want 3", base.Samples)
	}
}
