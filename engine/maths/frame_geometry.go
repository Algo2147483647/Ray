package maths

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
)

// NewFrameFromNormalInGeometry builds an orthonormal frame at point p
// whose normal direction is the geometry-projected version of n.
//
// For Euclidean/Klein this delegates to NewFrameFromNormal: the existing
// Gram-Schmidt in R^d already produces the right basis.
//
// For Spherical we first project both the normal and every candidate
// tangent direction into T_p S^3, so the resulting frame lives in the
// 3-dimensional tangent subspace embedded in R^4.
func NewFrameFromNormalInGeometry(g geometry.Geometry, p, n *mat.VecDense) (Frame, bool) {
	g = geometry.Get(g)
	if g.Name() != "spherical" {
		return NewFrameFromNormal(n)
	}

	// Project the normal into T_p, then orthonormalize.
	projected := mat.NewVecDense(n.Len(), nil)
	g.ProjectTangent(p, n, projected)
	if mat.Norm(projected, 2) <= 1e-12 {
		return Frame{}, false
	}
	Normalize(projected)

	dim := n.Len()
	tangents := make([]*mat.VecDense, 0, dim-2)
	for axis := 0; axis < dim && len(tangents) < dim-2; axis++ {
		candidate := mat.NewVecDense(dim, nil)
		candidate.SetVec(axis, 1)
		// Remove components along p (radial) and along the normal.
		candidate.AddScaledVec(candidate, -mat.Dot(candidate, p), p)
		candidate.AddScaledVec(candidate, -mat.Dot(candidate, projected), projected)
		for _, t := range tangents {
			candidate.AddScaledVec(candidate, -mat.Dot(candidate, t), t)
		}
		if mat.Norm(candidate, 2) <= 1e-12 {
			continue
		}
		tangents = append(tangents, Normalize(candidate))
	}
	if len(tangents) != dim-2 {
		return Frame{}, false
	}

	var tangent, bitangent *mat.VecDense
	if len(tangents) > 0 {
		tangent = tangents[0]
	}
	if len(tangents) > 1 {
		bitangent = tangents[1]
	}

	return Frame{
		Tangent:   tangent,
		Bitangent: bitangent,
		Normal:    projected,
		Tangents:  tangents,
	}, true
}
