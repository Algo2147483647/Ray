package camera

import (
	"fmt"
	"math"

	renderray "github.com/Algo2147483647/ray/engine/model/optics"
)

type Camera interface {
	GenerateRay(res *renderray.Ray, index ...int) *renderray.Ray // Generates a ray for a given pixel index.
}

type CameraType string

const (
	CameraType3D         CameraType = "3d"
	CameraTypeNDim       CameraType = "n_dim"
	CameraTypeHyperbolic CameraType = "hyperbolic"
	CameraTypeSpherical  CameraType = "spherical"
)

type CameraBase struct {
}

func (c *CameraBase) GenerateRay(ray *renderray.Ray, index ...int) *renderray.Ray {
	return &renderray.Ray{}
}

func frameHalfExtents(fieldOfViews []float64) (float64, float64, error) {
	if len(fieldOfViews) != 2 {
		return 0, 0, fmt.Errorf("camera field_of_views must contain vertical and horizontal FOV values, got %d", len(fieldOfViews))
	}
	verticalFOV := fieldOfViews[0]
	horizontalFOV := fieldOfViews[1]
	if verticalFOV <= 0 {
		return 0, 0, fmt.Errorf("camera field_of_views[0] must be > 0")
	}
	if horizontalFOV <= 0 {
		return 0, 0, fmt.Errorf("camera field_of_views[1] must be > 0")
	}
	halfHeight := math.Tan(verticalFOV * math.Pi / 180 / 2)
	halfWidth := math.Tan(horizontalFOV * math.Pi / 180 / 2)
	if halfHeight <= 0 || halfWidth <= 0 || math.IsNaN(halfHeight) || math.IsNaN(halfWidth) || math.IsInf(halfHeight, 0) || math.IsInf(halfWidth, 0) {
		return 0, 0, fmt.Errorf("camera field_of_views must produce finite positive frame extents")
	}
	return halfHeight, halfWidth, nil
}
