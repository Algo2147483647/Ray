package maths

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"gonum.org/v1/gonum/mat"
	"math"
)

// NewFrameFromNormalInGeometry builds a metric-orthonormal surface frame at p.
// n must be an intrinsic tangent-space normal vector (an ambient gradient
// should first be converted with Geometry.IntrinsicNormal).
func NewFrameFromNormalInGeometry(g geometry.Geometry, p, n *mat.VecDense) (Frame, bool) {
	if g == nil || p == nil || n == nil || p.Len() != n.Len() || n.Len() < 2 {
		return Frame{}, false
	}

	projected := mat.NewVecDense(n.Len(), nil)
	g.ProjectTangent(p, n, projected)
	if !normalizeVectorInGeometry(g, p, projected) {
		return Frame{}, false
	}

	dim := n.Len()
	tangentCount := dim - 1
	if g.Kind() == geometry.SphericalKind {
		tangentCount--
	}
	if tangentCount < 1 {
		return Frame{}, false
	}

	tangents := make([]*mat.VecDense, 0, tangentCount)
	for axis := 0; axis < dim && len(tangents) < tangentCount; axis++ {
		candidate := mat.NewVecDense(dim, nil)
		candidate.SetVec(axis, 1)
		g.ProjectTangent(p, candidate, candidate)
		subtractGeometryProjection(g, p, candidate, projected)
		for _, t := range tangents {
			subtractGeometryProjection(g, p, candidate, t)
		}
		if !normalizeVectorInGeometry(g, p, candidate) {
			continue
		}
		tangents = append(tangents, candidate)
	}
	if len(tangents) != tangentCount {
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
		Geometry:  g,
		Point:     mat.VecDenseCopyOf(p),
		Tangent:   tangent,
		Bitangent: bitangent,
		Normal:    projected,
		Tangents:  tangents,
	}, true
}

func subtractGeometryProjection(g geometry.Geometry, p, v, basis *mat.VecDense) {
	denominator := g.InnerProduct(p, basis, basis)
	if denominator <= 0 {
		return
	}
	scale := g.InnerProduct(p, v, basis) / denominator
	v.AddScaledVec(v, -scale, basis)
}

func normalizeVectorInGeometry(g geometry.Geometry, p, v *mat.VecDense) bool {
	n2 := g.InnerProduct(p, v, v)
	if n2 <= 1e-24 || math.IsNaN(n2) || math.IsInf(n2, 0) {
		return false
	}
	v.ScaleVec(1/math.Sqrt(n2), v)
	return true
}
