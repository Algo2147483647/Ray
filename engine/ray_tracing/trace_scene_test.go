package ray_tracing

import "testing"

func TestBuildRenderTilesCovers2DFilm(t *testing.T) {
	shape := []int{5, 3}
	tiles := buildRenderTiles(shape, 2, 2)
	seen := make(map[int]bool)

	for _, tile := range tiles {
		for y := tile.Y0; y < tile.Y1; y++ {
			for x := tile.X0; x < tile.X1; x++ {
				pixel := tile.pixelIndex(x, y, shape)
				if seen[pixel] {
					t.Fatalf("pixel %d rendered twice", pixel)
				}
				seen[pixel] = true
			}
		}
	}

	if len(seen) != shape[0]*shape[1] {
		t.Fatalf("expected %d covered pixels, got %d", shape[0]*shape[1], len(seen))
	}
}

func TestBuildRenderTilesFallsBackToLinearChunks(t *testing.T) {
	shape := []int{3, 2, 2}
	tiles := buildRenderTiles(shape, 2, 2)
	seen := make(map[int]bool)

	for _, tile := range tiles {
		for y := tile.Y0; y < tile.Y1; y++ {
			for x := tile.X0; x < tile.X1; x++ {
				pixel := tile.pixelIndex(x, y, shape)
				if seen[pixel] {
					t.Fatalf("pixel %d rendered twice", pixel)
				}
				seen[pixel] = true
			}
		}
	}

	if len(seen) != 12 {
		t.Fatalf("expected 12 covered pixels, got %d", len(seen))
	}
}
