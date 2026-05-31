package emission

import (
	"math"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/optics"
)

// CellPaletteShading selects how the cell label is turned into emitted radiance.
type CellPaletteShading int

const (
	// CellPaletteShadingEmission emits the cell color directly and returns it
	// to the camera; the path terminates on first hit (debug-style overlay).
	CellPaletteShadingEmission CellPaletteShading = iota
	// CellPaletteShadingBoundaryGrid emits the cell color in the interior of
	// the cell and a configurable grid color near cell boundaries. Useful for
	// visualising both cell identity and adjacency.
	CellPaletteShadingBoundaryGrid
)

// CellPalette emits a constant color chosen by the dominant axis of the
// geometric normal at the hit. For an axis-aligned shape (Cuboid /
// Hypercube / Hypercuboid) this means every cell of the shape gets its
// own color, which is the canonical way to disambiguate the 8 cells of a
// 4D hypercube in a single render. The palette is wrapped modulo its length
// so the same emitter works for 3D (6 faces), 4D (8 cells), 5D (10 cells)
// without any per-dimension configuration.
type CellPalette struct {
	Palette       []optics.Spectrum
	Shading       CellPaletteShading
	GridColor     optics.Spectrum // boundary grid color (only used by CellPaletteShadingBoundaryGrid)
	GridThickness float64         // boundary half-width in world units (only used by CellPaletteShadingBoundaryGrid)
}

// DefaultCellPalette is a high-contrast 8-color palette indexed as
// (-X,+X,-Y,+Y,-Z,+Z,-W,+W). Chosen for maximum perceptual separation after
// the engine's ACES + gamma 2.2 tone-map, so colors stay distinct in the
// final output even when channel values fall into the post-tone-map midtones.
var DefaultCellPalette = []optics.Spectrum{
	optics.NewSpectrum(1.00, 0.20, 0.20), // -X  red
	optics.NewSpectrum(0.20, 1.00, 0.20), // +X  green
	optics.NewSpectrum(0.20, 0.40, 1.00), // -Y  blue
	optics.NewSpectrum(1.00, 0.85, 0.20), // +Y  yellow
	optics.NewSpectrum(1.00, 0.30, 0.90), // -Z  magenta
	optics.NewSpectrum(0.20, 0.95, 0.95), // +Z  cyan
	optics.NewSpectrum(1.00, 0.55, 0.10), // -W  orange
	optics.NewSpectrum(0.92, 0.92, 0.92), // +W  near-white
}

// NewCellPalette returns a CellPalette emitter using the default palette and
// solid (non-grid) shading.
func NewCellPalette() CellPalette {
	return CellPalette{
		Palette: append([]optics.Spectrum(nil), DefaultCellPalette...),
		Shading: CellPaletteShadingEmission,
	}
}

func (c CellPalette) Emit(ctx bxdf.ShadingContext, _ maths.Direction) optics.Spectrum {
	palette := c.Palette
	if len(palette) == 0 {
		palette = DefaultCellPalette
	}

	axis, sign := dominantAxis(ctx.GeometricNormal)
	if axis < 0 {
		// No usable normal info: fall back to the first palette entry rather
		// than emitting zero, which would visually look like "missing material".
		return palette[0]
	}
	idx := (axis*2 + boolToInt(sign > 0)) % len(palette)
	cellColor := palette[idx]

	if c.Shading != CellPaletteShadingBoundaryGrid {
		return cellColor
	}

	// Boundary grid: emit GridColor whenever the hit point is close to *any*
	// of the cell's non-dominant edges (cells share faces along non-dominant
	// axes, so the gap to those axes' AABB limits is the natural metric).
	if !boundaryHit(ctx, axis, c.GridThickness) {
		return cellColor
	}
	if c.GridColor.IsZero() {
		return optics.NewSpectrum(1, 1, 1)
	}
	return c.GridColor
}

func (CellPalette) IsDelta() bool { return false }

// dominantAxis returns the axis with the largest absolute component of the
// normal, and the sign of that component. Returns (-1, 0) when the input is
// empty or numerically zero.
func dominantAxis(n maths.Direction) (int, float64) {
	if n.Len() == 0 {
		return -1, 0
	}
	best := -1
	bestMag := 0.0
	for i := 0; i < n.Len(); i++ {
		m := math.Abs(n.Component(i))
		if m > bestMag {
			best = i
			bestMag = m
		}
	}
	if best < 0 || bestMag == 0 {
		return -1, 0
	}
	return best, sign(n.Component(best))
}

func boundaryHit(ctx bxdf.ShadingContext, axis int, thickness float64) bool {
	if thickness <= 0 {
		return false
	}
	if ctx.HitPoint.Len() == 0 || ctx.HitObjectAABBMin.Len() == 0 || ctx.HitObjectAABBMax.Len() == 0 {
		return false
	}
	for i := 0; i < ctx.HitPoint.Len(); i++ {
		if i == axis {
			// The dominant axis is pinned to one face of the AABB; ignore it.
			continue
		}
		p := ctx.HitPoint.Component(i)
		lo := ctx.HitObjectAABBMin.Component(i)
		hi := ctx.HitObjectAABBMax.Component(i)
		if math.Min(p-lo, hi-p) <= thickness {
			return true
		}
	}
	return false
}

func sign(v float64) float64 {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
