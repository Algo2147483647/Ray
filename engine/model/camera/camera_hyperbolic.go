package camera

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	renderray "github.com/Algo2147483647/ray/engine/model/optics"
)

// HyperbolicCamera is a Camera3D whose generated rays are tagged with the
// Klein H^3 geometry. Ray generation logic is otherwise identical: in the
// Klein model, the chord from the camera position with any embedded
// direction is the geodesic.
type HyperbolicCamera struct {
	Camera3D
}

func NewHyperbolicCamera() *HyperbolicCamera { return &HyperbolicCamera{} }

func (c *HyperbolicCamera) GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray {
	res = c.Camera3D.GenerateRay(res, index...)
	res.Geometry = geometry.Klein()
	return res
}
