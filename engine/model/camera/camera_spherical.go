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

// SphericalCamera lives on S^3 embedded in R^4. Position is a unit vector;
// Coordinates are R^4 basis vectors projected into T_p and orthonormalized at
// Prepare time.
type SphericalCamera struct {
	Position     *mat.VecDense
	Coordinates  []*mat.VecDense // Camera basis vectors: forward, right, up.
	FieldOfViews []float64       // Vertical and horizontal field-of-view angles in degrees.

	orthonormalCoordinates []*mat.VecDense
	halfWidth              float64
	halfHeight             float64
	prepared               bool
}

func NewSphericalCamera() *SphericalCamera { return &SphericalCamera{} }

func (c *SphericalCamera) Prepare() error {
	if c.Position == nil || len(c.Coordinates) != 3 {
		return fmt.Errorf("spherical camera requires position and three basis vectors")
	}
	halfHeight := math.Tan(c.FieldOfViews[0] * math.Pi / 180 / 2)
	halfWidth := math.Tan(c.FieldOfViews[1] * math.Pi / 180 / 2)

	g := geometry.Spherical()

	// Renormalize position onto S^3.
	pos := mat.VecDenseCopyOf(c.Position)
	maths.Normalize(pos)
	c.Position = pos

	// Project the authored basis into T_p, then orthonormalize.
	fwd := mat.NewVecDense(4, nil)
	g.ProjectTangent(c.Position, c.Coordinates[0], fwd)
	if mat.Norm(fwd, 2) == 0 {
		return fmt.Errorf("forward direction collapses in T_p")
	}
	maths.Normalize(fwd)

	right := mat.NewVecDense(4, nil)
	g.ProjectTangent(c.Position, c.Coordinates[1], right)
	right.AddScaledVec(right, -mat.Dot(right, fwd), fwd)
	if mat.Norm(right, 2) == 0 {
		return fmt.Errorf("right direction collapses after orthogonalization")
	}
	maths.Normalize(right)

	up := mat.NewVecDense(4, nil)
	g.ProjectTangent(c.Position, c.Coordinates[2], up)
	up.AddScaledVec(up, -mat.Dot(up, fwd), fwd)
	up.AddScaledVec(up, -mat.Dot(up, right), right)
	if mat.Norm(up, 2) == 0 {
		return fmt.Errorf("up direction collapses after orthogonalization")
	}
	maths.Normalize(up)
	c.orthonormalCoordinates = []*mat.VecDense{fwd, right, up}

	c.halfHeight = halfHeight
	c.halfWidth = halfWidth
	c.prepared = true
	return nil
}

// orthogonalInTangent returns a unit vector in T_p orthogonal to a and b.
// p, a, b are assumed orthonormal already (p radial; a, b in T_p, mutually
// orthonormal). Probes each coordinate axis, orthogonalizes, picks first
// non-degenerate.
func orthogonalInTangent(p, a, b *mat.VecDense) *mat.VecDense {
	for axis := 0; axis < 4; axis++ {
		cand := mat.NewVecDense(4, nil)
		cand.SetVec(axis, 1)
		cand.AddScaledVec(cand, -mat.Dot(cand, p), p)
		cand.AddScaledVec(cand, -mat.Dot(cand, a), a)
		cand.AddScaledVec(cand, -mat.Dot(cand, b), b)
		if mat.Norm(cand, 2) > 1e-9 {
			maths.Normalize(cand)
			return cand
		}
	}
	return nil
}

func (c *SphericalCamera) GenerateRay(res *renderray.Ray, film *Film, index ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()
	if !c.prepared {
		if err := c.Prepare(); err != nil {
			panic(err)
		}
	}
	width, height := film.Shape[0], film.Shape[1]

	row, col := index[0], index[1]
	u := 2*(float64(row)+rand.Float64())/float64(width) - 1
	v := 2*(float64(col)+rand.Float64())/float64(height) - 1

	if res.Origin.Len() != 4 {
		res.Origin = mat.NewVecDense(4, nil)
	}
	if res.Direction.Len() != 4 {
		res.Direction = mat.NewVecDense(4, nil)
	}

	res.Origin.CopyVec(c.Position)
	res.Direction.CopyVec(c.orthonormalCoordinates[0])
	res.Direction.AddScaledVec(res.Direction, u*c.halfWidth, c.orthonormalCoordinates[1])
	res.Direction.AddScaledVec(res.Direction, -v*c.halfHeight, c.orthonormalCoordinates[2])
	// Direction already lives in T_p (sum of T_p vectors). Normalize.
	maths.Normalize(res.Direction)

	res.Geometry = geometry.Spherical()
	return res
}
