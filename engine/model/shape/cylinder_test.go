package shape

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/maths"
	"gonum.org/v1/gonum/mat"
)

func TestFiniteCylinderIntersectHitsSide(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	interaction, ok := cylinder.IntersectAffine(
		mat.NewVecDense(3, []float64{2, 0, 0}),
		mat.NewVecDense(3, []float64{-1, 0, 0}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)

	if !ok || math.Abs(interaction.Distance-1) > 1e-9 {
		t.Fatalf("unexpected side hit: ok=%v distance=%f want 1", ok, interaction.Distance)
	}
}

func TestFiniteCylinderIntersectHitsCap(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	interaction, ok := cylinder.IntersectAffine(
		mat.NewVecDense(3, []float64{0.5, 0, 4}),
		mat.NewVecDense(3, []float64{0, 0, -1}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)

	if !ok || math.Abs(interaction.Distance-2) > 1e-9 {
		t.Fatalf("unexpected cap hit: ok=%v distance=%f want 2", ok, interaction.Distance)
	}
}

func TestFiniteCylinderIntersectRejectsBeyondHeight(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	_, ok := cylinder.IntersectAffine(
		mat.NewVecDense(3, []float64{2, 0, 3}),
		mat.NewVecDense(3, []float64{-1, 0, 0}),
		NewIntersectOptions(1e-6, math.MaxFloat64),
	)

	if ok {
		t.Fatal("expected miss beyond cylinder height")
	}
}

func TestFiniteCylinderNormal(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{0, 0, 1}),
		1,
		4,
	)

	sideNormal := cylinder.GetNormalVector(mat.NewVecDense(3, []float64{1, 0, 0}), mat.NewVecDense(3, nil))
	topNormal := cylinder.GetNormalVector(mat.NewVecDense(3, []float64{0.5, 0, 2}), mat.NewVecDense(3, nil))

	if math.Abs(sideNormal.AtVec(0)-1) > 1e-9 || math.Abs(sideNormal.AtVec(2)) > 1e-9 {
		t.Fatalf("unexpected side normal: %v", sideNormal.RawVector().Data)
	}
	if math.Abs(topNormal.AtVec(2)-1) > 1e-9 {
		t.Fatalf("unexpected top normal: %v", topNormal.RawVector().Data)
	}
}

func TestFiniteCylinderBuildBoundingBox(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{1, 2, 3}),
		mat.NewVecDense(3, []float64{0, 0, 2}),
		1,
		4,
	)

	pmin, pmax := cylinder.BuildBoundingBox()
	wantMin := []float64{0, 1, 1}
	wantMax := []float64{2, 3, 5}

	for i := 0; i < 3; i++ {
		if math.Abs(pmin.AtVec(i)-wantMin[i]) > 1e-9 || math.Abs(pmax.AtVec(i)-wantMax[i]) > 1e-9 {
			t.Fatalf("unexpected bbox axis %d: got [%f, %f] want [%f, %f]", i, pmin.AtVec(i), pmax.AtVec(i), wantMin[i], wantMax[i])
		}
	}
}

func TestFiniteCylinderSampleSurfaceUsesExactAreaPartition(t *testing.T) {
	cylinder := NewFiniteCylinder(
		mat.NewVecDense(3, []float64{1, -2, 4}),
		mat.NewVecDense(3, []float64{1, 1, 1}),
		0.5,
		3,
	)
	totalArea := cylinder.SurfaceArea()
	sideArea := 2 * math.Pi * cylinder.R * cylinder.Height
	capArea := math.Pi * cylinder.R * cylinder.R

	tests := []struct {
		name         string
		u            float64
		wantAxis     float64
		wantRadius   float64
		wantNormalAx float64
	}{
		{name: "side", u: 0.5 * sideArea / totalArea, wantAxis: 0, wantRadius: cylinder.R, wantNormalAx: 0},
		{name: "top cap", u: (sideArea + 0.25*capArea) / totalArea, wantAxis: cylinder.Height / 2, wantRadius: cylinder.R * 0.5, wantNormalAx: 1},
		{name: "bottom cap", u: (sideArea + capArea + 0.25*capArea) / totalArea, wantAxis: -cylinder.Height / 2, wantRadius: cylinder.R * 0.5, wantNormalAx: -1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sample, ok := cylinder.SampleSurface(maths.Sample2D{U: test.u, V: 0.375})
			if !ok {
				t.Fatal("expected cylinder surface sample")
			}
			offset := mat.NewVecDense(3, nil)
			offset.SubVec(sample.Point, cylinder.Center)
			axisDistance := mat.Dot(offset, cylinder.Axis)
			radial := mat.NewVecDense(3, nil)
			radial.AddScaledVec(offset, -axisDistance, cylinder.Axis)
			if math.Abs(axisDistance-test.wantAxis) > 1e-12 {
				t.Fatalf("axis distance = %g, want %g", axisDistance, test.wantAxis)
			}
			if math.Abs(mat.Norm(radial, 2)-test.wantRadius) > 1e-12 {
				t.Fatalf("radius = %g, want %g", mat.Norm(radial, 2), test.wantRadius)
			}
			if math.Abs(mat.Dot(sample.Normal, cylinder.Axis)-test.wantNormalAx) > 1e-12 {
				t.Fatalf("normal axis component = %g, want %g", mat.Dot(sample.Normal, cylinder.Axis), test.wantNormalAx)
			}
			if math.Abs(mat.Norm(sample.Normal, 2)-1) > 1e-12 {
				t.Fatalf("normal length = %g, want 1", mat.Norm(sample.Normal, 2))
			}
			if math.Abs(sample.PDFArea-1/totalArea) > 1e-12 {
				t.Fatalf("area PDF = %g, want %g", sample.PDFArea, 1/totalArea)
			}
		})
	}
}

func TestFiniteCylinderSampleSurfaceRejectsInvalidGeometry(t *testing.T) {
	invalid := []*FiniteCylinder{
		nil,
		NewFiniteCylinder(mat.NewVecDense(4, nil), mat.NewVecDense(4, []float64{0, 0, 0, 1}), 1, 1),
		{Center: mat.NewVecDense(3, nil), Axis: mat.NewVecDense(3, []float64{0, 0, 1}), R: 0, Height: 1},
		{Center: mat.NewVecDense(3, nil), Axis: mat.NewVecDense(3, nil), R: 1, Height: 1},
	}
	for index, cylinder := range invalid {
		if cylinder != nil && cylinder.SurfaceArea() != 0 {
			t.Fatalf("invalid cylinder %d has nonzero area", index)
		}
		if _, ok := cylinder.SampleSurface(maths.Sample2D{U: 0.5, V: 0.5}); ok {
			t.Fatalf("invalid cylinder %d was sampleable", index)
		}
	}
}
