package ray_tracing

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/model/film"
)

func TestBuildTileCoordinatesForWindowsKeepsOnlyRequestedPixels(t *testing.T) {
	shape := []int{10, 10}
	windows := []film.PixelWindow{
		{Min: []int{2, 3}, Max: []int{5, 5}},
		{Min: []int{4, 4}, Max: []int{7, 6}},
	}

	tiles, total := buildTileCoordinatesForWindows(shape, windows, 2, 2)

	if total != 11 {
		t.Fatalf("expected 11 unique pixels, got %d", total)
	}

	rendered := make([]bool, 100)
	for _, tile := range tiles {
		for y := tile.Y0; y < tile.Y1; y++ {
			for x := tile.X0; x < tile.X1; x++ {
				rendered[tile.pixelIndex(x, y, shape)] = true
			}
		}
	}

	for y := 0; y < shape[1]; y++ {
		for x := 0; x < shape[0]; x++ {
			want := (x >= 2 && x < 5 && y >= 3 && y < 5) ||
				(x >= 4 && x < 7 && y >= 4 && y < 6)
			if rendered[y*shape[0]+x] != want {
				t.Fatalf("pixel (%d,%d) rendered=%v, want %v", x, y, rendered[y*shape[0]+x], want)
			}
		}
	}
}

func TestBuildTileCoordinatesForWindowsWithoutWindowsUsesFullFilm(t *testing.T) {
	tiles, total := buildTileCoordinatesForWindows([]int{4, 3}, nil, 2, 2)

	if total != 12 {
		t.Fatalf("expected full film total, got %d", total)
	}
	if len(tiles) == 0 {
		t.Fatal("expected full-film tiles")
	}
}
