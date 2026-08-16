package camera

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths/geometry"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

// HyperbolicCamera is a Camera3D whose local camera frame is orthonormalized
// under the Klein H^3 metric at the camera position. In the Klein model,
// geodesics are Euclidean chords, but angles and field-of-view live in the
// tangent metric g_p rather than in the ambient Euclidean dot product.
type HyperbolicCamera struct {
	Camera3D
	hyperbolicPrepared bool
}

func NewHyperbolicCamera() *HyperbolicCamera { return &HyperbolicCamera{} }

func (c *HyperbolicCamera) Prepare() error {
	if c.Position == nil {
		return fmt.Errorf("camera position is not configured")
	} else if len(c.Coordinates) != 3 {
		return fmt.Errorf("camera coordinates must contain forward, right, and up vectors")
	} else if mat.Dot(c.Position, c.Position) >= 1 {
		return fmt.Errorf("hyperbolic camera position must be inside the Klein unit ball")
	}
	halfHeight := math.Tan(c.FieldOfViews[0] * math.Pi / 180 / 2)
	halfWidth := math.Tan(c.FieldOfViews[1] * math.Pi / 180 / 2)

	g := geometry.Klein()
	fwd := mat.VecDenseCopyOf(c.Coordinates[0])
	if !normalizeInGeometry(g, c.Position, fwd) {
		return fmt.Errorf("camera direction has zero Klein length")
	}

	right := mat.VecDenseCopyOf(c.Coordinates[1])
	orthogonalizeInGeometry(g, c.Position, right, fwd)
	if !normalizeInGeometry(g, c.Position, right) {
		return fmt.Errorf("could not construct right vector in Klein tangent metric")
	}

	up := mat.VecDenseCopyOf(c.Coordinates[2])
	orthogonalizeInGeometry(g, c.Position, up, fwd)
	orthogonalizeInGeometry(g, c.Position, up, right)
	if !normalizeInGeometry(g, c.Position, up) {
		return fmt.Errorf("could not construct up vector in Klein tangent metric")
	}

	c.orthonormalCoordinates = []*mat.VecDense{fwd, right, up}

	c.halfHeight = halfHeight
	c.halfWidth = halfWidth
	c.prepared = true
	c.hyperbolicPrepared = true
	return nil
}

func (c *HyperbolicCamera) GenerateRay(res *renderray.Ray, film *Film, index ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()

	if !c.hyperbolicPrepared {
		if err := c.Prepare(); err != nil {
			panic(err)
		}
	}
	width, height := film.Shape[0], film.Shape[1]

	row, col := index[0], index[1]
	u := 2*(float64(row)+rand.Float64())/float64(width) - 1
	v := 2*(float64(col)+rand.Float64())/float64(height) - 1

	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(c.orthonormalCoordinates[0])
	res.Direction.AddScaledVec(res.Direction, u*c.halfWidth, c.orthonormalCoordinates[1])
	res.Direction.AddScaledVec(res.Direction, -v*c.halfHeight, c.orthonormalCoordinates[2])
	normalizeInGeometry(geometry.Klein(), c.Position, res.Direction)
	res.Geometry = geometry.Klein()
	return res
}

func orthogonalizeInGeometry(g geometry.Geometry, p, v, basis *mat.VecDense) {
	den := g.InnerProduct(p, basis, basis)
	if den == 0 || math.IsNaN(den) || math.IsInf(den, 0) {
		return
	}
	scale := g.InnerProduct(p, v, basis) / den
	v.AddScaledVec(v, -scale, basis)
}

func normalizeInGeometry(g geometry.Geometry, p, v *mat.VecDense) bool {
	n2 := g.InnerProduct(p, v, v)
	if n2 <= 0 || math.IsNaN(n2) || math.IsInf(n2, 0) {
		return false
	}
	v.ScaleVec(1/math.Sqrt(n2), v)
	return true
}
