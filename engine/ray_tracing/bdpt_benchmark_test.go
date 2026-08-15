package ray_tracing

import (
	"fmt"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/bsdf"
	"github.com/Algo2147483647/ray/engine/model/material/bxdf"
	"github.com/Algo2147483647/ray/engine/model/object"
	"github.com/Algo2147483647/ray/engine/model/optics"
	"gonum.org/v1/gonum/mat"
)

func BenchmarkBDPTMISByDepth(b *testing.B) {
	for _, depth := range []int{4, 8, 12, 16} {
		b.Run(fmt.Sprintf("depth_%d", depth), func(b *testing.B) {
			light, cameraPath, li, ci := benchmarkBDPTPaths(b, depth)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = bdptMISWeight(light, cameraPath, li, ci)
			}
		})
	}
}

func benchmarkBDPTPaths(tb testing.TB, vertexCount int) ([]bdptVertex, []bdptVertex, int, int) {
	tb.Helper()
	if vertexCount < 4 {
		vertexCount = 4
	}
	lambert := &object.Object{Material: &material.Material{
		Surface: bsdf.NewSingle(bxdf.NewLambert(optics.ConstantSpectrum(0.8))),
	}}
	points := make([]*mat.VecDense, vertexCount)
	for i := range points {
		points[i] = mat.NewVecDense(3, []float64{
			float64(i), 0, 0.15 * float64((i%2)*2-1),
		})
	}
	complete := make([]bdptVertex, vertexCount)
	for i := range complete {
		var normal *mat.VecDense
		switch i {
		case 0:
			normal = directionBetween(points[i], points[i+1])
		case vertexCount - 1:
			normal = directionBetween(points[i], points[i-1])
		default:
			toPrevious := directionBetween(points[i], points[i-1])
			toNext := directionBetween(points[i], points[i+1])
			normal = mat.NewVecDense(3, nil)
			normal.AddVec(toPrevious, toNext)
			if mat.Norm(normal, 2) < 1e-6 {
				normal = mat.NewVecDense(3, []float64{0, 0, 1})
			} else {
				normal.ScaleVec(1/mat.Norm(normal, 2), normal)
			}
		}
		frame, ok := maths.NewFrameFromNormal(normal)
		if !ok {
			tb.Fatal("failed to build benchmark frame")
		}
		complete[i] = bdptVertex{
			Point: points[i], GeometricNormal: normal, Frame: frame,
			WoLocal: maths.NewDirection(0, 0, 1), Object: lambert,
		}
	}
	complete[0].LightEndpoint = true
	complete[0].PDFFwdArea = 0.25

	split := vertexCount / 2
	lightPath := append([]bdptVertex(nil), complete[:split]...)
	cameraPath := make([]bdptVertex, vertexCount-split)
	for i := range cameraPath {
		cameraPath[i] = complete[vertexCount-1-i]
	}
	return lightPath, cameraPath, len(lightPath) - 1, len(cameraPath) - 1
}
