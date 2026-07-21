package camera

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/Algo2147483647/ray/engine/maths"
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
	} else if c.Direction == nil {
		return fmt.Errorf("camera direction is not configured")
	} else if c.Up == nil {
		return fmt.Errorf("camera up vector is not configured")
	} else if c.Width <= 0 {
		return fmt.Errorf("camera width must be > 0")
	} else if c.Height <= 0 {
		return fmt.Errorf("camera height must be > 0")
	} else if mat.Norm(c.Direction, 2) == 0 {
		return fmt.Errorf("camera direction must not be zero")
	} else if mat.Norm(c.Up, 2) == 0 {
		return fmt.Errorf("camera up vector must not be zero")
	} else if mat.Dot(c.Position, c.Position) >= 1 {
		return fmt.Errorf("hyperbolic camera position must be inside the Klein unit ball")
	}
	halfHeight, halfWidth, err := frameHalfExtents(c.FieldOfViews)
	if err != nil {
		return err
	}

	g := geometry.Klein()
	fwd := mat.VecDenseCopyOf(c.Direction)
	if !normalizeInGeometry(g, c.Position, fwd) {
		return fmt.Errorf("camera direction has zero Klein length")
	}

	up := mat.VecDenseCopyOf(c.Up)
	orthogonalizeInGeometry(g, c.Position, up, fwd)
	if !normalizeInGeometry(g, c.Position, up) {
		return fmt.Errorf("camera direction and up vector must not be parallel")
	}

	right := maths.Cross2(fwd, up)
	orthogonalizeInGeometry(g, c.Position, right, fwd)
	orthogonalizeInGeometry(g, c.Position, right, up)
	if !normalizeInGeometry(g, c.Position, right) {
		return fmt.Errorf("could not construct right vector in Klein tangent metric")
	}

	c.dir = fwd
	c.up = up
	c.right = right

	c.halfHeight = halfHeight
	c.halfWidth = halfWidth
	c.invWidth2 = 2 / float64(c.Width)
	c.invHeight2 = 2 / float64(c.Height)
	c.prepared = true
	c.hyperbolicPrepared = true
	return nil
}

func (c *HyperbolicCamera) GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()

	if !c.hyperbolicPrepared {
		if err := c.Prepare(); err != nil {
			panic(err)
		}
	}

	row, col := index[0], index[1]
	u := (float64(row)+rand.Float64())*c.invWidth2 - 1
	v := (float64(col)+rand.Float64())*c.invHeight2 - 1

	res.Origin.CloneFromVec(c.Position)
	res.Direction.CloneFromVec(c.dir)
	res.Direction.AddScaledVec(res.Direction, u*c.halfWidth, c.right)
	res.Direction.AddScaledVec(res.Direction, -v*c.halfHeight, c.up)
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
