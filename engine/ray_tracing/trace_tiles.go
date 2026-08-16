package ray_tracing

import "github.com/Algo2147483647/ray/engine/model/camera"

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

func (h *Handler) traceTile(
	kernel pixelKernel,
	session *RenderSession,
	tile TileCoordinate,
) int64 {
	var rendered int64

	for y := tile.Y0; y < tile.Y1; y++ {
		for x := tile.X0; x < tile.X1; x++ {
			film := session.Context.Camera.GetFilm()
			pixel := tile.pixelIndex(x, y, film.Shape)
			coords := film.SpectralBins[0].GetCoordinates(pixel)

			h.tracePixel(kernel, session, pixel, coords...)

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

func buildTileCoordinatesForWindows(shape []int, windows []camera.PixelWindow, tileWidth, tileHeight int) ([]TileCoordinate, int64) {
	total := shapeElementCount(shape)
	if len(windows) == 0 {
		return buildTileCoordinates(shape, tileWidth, tileHeight), int64(total)
	}

	if tileWidth <= 0 {
		tileWidth = defaultTileSize
	}
	if tileHeight <= 0 {
		tileHeight = defaultTileSize
	}

	normalized, err := camera.NormalizePixelWindows(windows, shape)
	if err != nil {
		return nil, 0
	}
	mask, pixels := buildPixelWindowMask(shape, normalized)
	return buildMaskedLinearRenderTiles(mask, tileWidth*tileHeight), pixels
}

func buildLinearRenderTiles(shape []int, chunkSize int) []TileCoordinate {
	total := shapeElementCount(shape)

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

func buildMaskedLinearRenderTiles(mask []bool, chunkSize int) []TileCoordinate {
	if chunkSize <= 0 {
		chunkSize = defaultTileSize * defaultTileSize
	}

	tiles := []TileCoordinate{}
	for start := 0; start < len(mask); {
		if !mask[start] {
			start++
			continue
		}

		end := start + 1
		for end < len(mask) && mask[end] {
			end++
		}

		for chunkStart := start; chunkStart < end; chunkStart += chunkSize {
			chunkEnd := min(chunkStart+chunkSize, end)
			tiles = append(tiles, TileCoordinate{
				X0: chunkStart,
				X1: chunkEnd,
				Y0: 0,
				Y1: 1,
			})
		}
		start = end
	}
	return tiles
}

func buildPixelWindowMask(shape []int, windows []camera.PixelWindow) ([]bool, int64) {
	total := shapeElementCount(shape)
	mask := make([]bool, total)
	strides := pixelStrides(shape)
	var pixels int64

	var mark func(window camera.PixelWindow, dim, index int)
	mark = func(window camera.PixelWindow, dim, index int) {
		if dim == len(shape) {
			if !mask[index] {
				mask[index] = true
				pixels++
			}
			return
		}

		for coord := window.Min[dim]; coord < window.Max[dim]; coord++ {
			mark(window, dim+1, index+coord*strides[dim])
		}
	}

	for _, window := range windows {
		mark(window, 0, 0)
	}
	return mask, pixels
}

func pixelStrides(shape []int) []int {
	strides := make([]int, len(shape))
	stride := 1
	for i, dim := range shape {
		strides[i] = stride
		stride *= dim
	}
	return strides
}

func shapeElementCount(shape []int) int {
	total := 1
	for _, dim := range shape {
		total *= dim
	}
	return total
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
