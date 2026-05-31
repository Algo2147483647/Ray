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
// Forward and Up are R^4 vectors interpreted as tangent vectors at Position
// (they are projected into T_p and orthonormalized at Prepare time).
type SphericalCamera struct {
	CameraBase
	Position    *mat.VecDense
	Forward     *mat.VecDense
	Up          *mat.VecDense
	Width       int
	Height      int
	FieldOfView float64 // degrees
	AspectRatio float64

	forward    *mat.VecDense
	up         *mat.VecDense
	right      *mat.VecDense
	halfWidth  float64
	halfHeight float64
	invWidth2  float64
	invHeight2 float64
	prepared   bool
}

func NewSphericalCamera() *SphericalCamera { return &SphericalCamera{} }

func (c *SphericalCamera) Prepare() error {
	if c.Position == nil || c.Forward == nil || c.Up == nil {
		return fmt.Errorf("spherical camera requires position, forward, up")
	}
	if c.Width <= 0 || c.Height <= 0 || c.FieldOfView <= 0 || c.AspectRatio <= 0 {
		return fmt.Errorf("spherical camera requires positive width, height, fov, aspect ratio")
	}

	g := geometry.Spherical()

	// Renormalize position onto S^3.
	pos := mat.VecDenseCopyOf(c.Position)
	maths.Normalize(pos)
	c.Position = pos

	// Project Forward and Up into T_p, then orthonormalize.
	fwd := mat.NewVecDense(4, nil)
	g.ProjectTangent(c.Position, c.Forward, fwd)
	if mat.Norm(fwd, 2) == 0 {
		return fmt.Errorf("forward direction collapses in T_p")
	}
	maths.Normalize(fwd)

	up := mat.NewVecDense(4, nil)
	g.ProjectTangent(c.Position, c.Up, up)
	up.AddScaledVec(up, -mat.Dot(up, fwd), fwd)
	if mat.Norm(up, 2) == 0 {
		return fmt.Errorf("up direction collapses after orthogonalization")
	}
	maths.Normalize(up)

	// Right = the third tangent direction: orthogonal in T_p to both fwd and up
	// and to Position. Find a coordinate axis with the smallest projection and
	// Gram-Schmidt against (Position, fwd, up).
	right := orthogonalInTangent(c.Position, fwd, up)
	if right == nil {
		return fmt.Errorf("could not construct right vector in T_p")
	}
	c.forward, c.up, c.right = fwd, up, right

	fovRad := c.FieldOfView * math.Pi / 180
	c.halfHeight = math.Tan(fovRad / 2)
	c.halfWidth = c.AspectRatio * c.halfHeight
	c.invWidth2 = 2 / float64(c.Width)
	c.invHeight2 = 2 / float64(c.Height)
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

func (c *SphericalCamera) GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray {
	if res == nil {
		res = &renderray.Ray{}
	}
	res.Init()
	if !c.prepared {
		if err := c.Prepare(); err != nil {
			panic(err)
		}
	}

	row, col := index[0], index[1]
	u := (float64(row)+rand.Float64())*c.invWidth2 - 1
	v := (float64(col)+rand.Float64())*c.invHeight2 - 1

	if res.Origin.Len() != 4 {
		res.Origin = mat.NewVecDense(4, nil)
	}
	if res.Direction.Len() != 4 {
		res.Direction = mat.NewVecDense(4, nil)
	}

	res.Origin.CopyVec(c.Position)
	res.Direction.CopyVec(c.forward)
	res.Direction.AddScaledVec(res.Direction, u*c.halfWidth, c.right)
	res.Direction.AddScaledVec(res.Direction, -v*c.halfHeight, c.up)
	// Direction already lives in T_p (sum of T_p vectors). Normalize.
	maths.Normalize(res.Direction)

	res.Geometry = geometry.Spherical()
	return res
}
