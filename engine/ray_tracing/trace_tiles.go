package ray_tracing

import (
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/object"
)

type TileCoordinate struct {
	X0 int
	X1 int
	Y0 int
	Y1 int
}

func (t TileCoordinate) pixelIndex(x, y int, shape []int) int {
	if len(shape) == 2 {
		return y*shape[0] + x
	}

	return x
}

func (h *Handler) TraceTile(
	renderCamera camera.Camera,
	objectTree *object.ObjectTree,
	film *camera.Film,
	samples int64,
	tile TileCoordinate,
) int64 {
	var rendered int64

	for y := tile.Y0; y < tile.Y1; y++ {
		for x := tile.X0; x < tile.X1; x++ {
			pixel := tile.pixelIndex(x, y, film.Data[0].Shape)
			coords := film.Data[0].GetCoordinates(pixel)

			h.TracePixel(renderCamera, objectTree, film, samples, pixel, coords...)

			rendered++
		}
	}

	return rendered
}

func buildTileCoordinates(shape []int, tileWidth, tileHeight int) []TileCoordinate {
	if tileWidth <= 0 {
		tileWidth = defaultTileSize
	}

	if tileHeight <= 0 {
		tileHeight = defaultTileSize
	}

	if len(shape) != 2 {
		return buildLinearRenderTiles(shape, tileWidth*tileHeight)
	}
	return build2DRenderTiles(shape[0], shape[1], tileWidth, tileHeight)
}

func buildLinearRenderTiles(shape []int, chunkSize int) []TileCoordinate {
	total := 1
	for _, dim := range shape {
		total *= dim
	}

	tiles := make([]TileCoordinate, 0, (total+chunkSize-1)/chunkSize)

	for start := 0; start < total; start += chunkSize {
		end := min(start+chunkSize, total)

		tiles = append(tiles, TileCoordinate{
			X0: start,
			X1: end,
			Y0: 0,
			Y1: 1,
		})
	}

	return tiles
}

func build2DRenderTiles(width, height, tileWidth, tileHeight int) []TileCoordinate {
	cols := (width + tileWidth - 1) / tileWidth
	rows := (height + tileHeight - 1) / tileHeight
	tiles := make([]TileCoordinate, 0, cols*rows)

	for y0 := 0; y0 < height; y0 += tileHeight {
		y1 := min(y0+tileHeight, height)

		for x0 := 0; x0 < width; x0 += tileWidth {
			x1 := min(x0+tileWidth, width)

			tiles = append(tiles, TileCoordinate{
				X0: x0,
				X1: x1,
				Y0: y0,
				Y1: y1,
			})
		}
	}

	return tiles
}
