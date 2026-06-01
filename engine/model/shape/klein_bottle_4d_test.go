package shape

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

func TestKleinBottle4DSDFOnBaseAndTubeBoundary(t *testing.T) {
	k := NewKleinBottle4D(mat.NewVecDense(4, []float64{0, 0, 0, 0}), 1.5, 0.5, 0.06)

	base := mat.NewVecDense(4, []float64{2, 0, 0, 0})
	if got := k.SDF(base); math.Abs(got+k.Thickness) > 1e-9 {
		t.Fatalf("base surface SDF = %.15f, want -thickness", got)
	}

	boundary := mat.NewVecDense(4, []float64{2 + k.Thickness, 0, 0, 0})
	if got := k.SDF(boundary); math.Abs(got) > 1e-9 {
		t.Fatalf("tube boundary SDF = %.15f, want 0", got)
	}

	outside := mat.NewVecDense(4, []float64{2 + k.Thickness + 0.125, 0, 0, 0})
	if got := k.SDF(outside); math.Abs(got-0.125) > 1e-9 {
		t.Fatalf("outside SDF = %.15f, want 0.125", got)
	}
}

func TestKleinBottle4DWrapsUVAcrossSeam(t *testing.T) {
	u, v := wrapKleinUV(2*math.Pi+0.25, 0.75)

	if math.Abs(u-0.25) > 1e-12 {
		t.Fatalf("wrapped u = %.15f, want 0.25", u)
	}
	if math.Abs(v-(2*math.Pi-0.75)) > 1e-12 {
		t.Fatalf("wrapped v = %.15f, want 2*pi-0.75", v)
	}
}

func TestKleinBottle4DIntersectRangeHitsTubeBoundary(t *testing.T) {
	k := NewKleinBottle4D(mat.NewVecDense(4, []float64{0, 0, 0, 0}), 1.5, 0.5, 0.06)
	k.marchEps = 1e-8
	k.minStep = 1e-8

	start := mat.NewVecDense(4, []float64{2 + k.Thickness + 0.25, 0, 0, 0})
	dir := mat.NewVecDense(4, []float64{-1, 0, 0, 0})

	interaction, ok := k.IntersectRange(start, dir, 0, 1)
	if !ok {
		t.Fatal("expected ray to hit Klein bottle tube")
	}
	if math.Abs(interaction.Distance-0.25) > 1e-5 {
		t.Fatalf("hit distance = %.15f, want 0.25", interaction.Distance)
	}
	if got := k.SDF(interaction.Point); math.Abs(got) > 1e-5 {
		t.Fatalf("hit point SDF = %.15f, want near 0", got)
	}
	if interaction.Point.Len() != 4 || interaction.GeometricNormal.Len() != 4 {
		t.Fatalf("expected 4D interaction, got point=%d normal=%d", interaction.Point.Len(), interaction.GeometricNormal.Len())
	}
	if interaction.GeometricNormal.AtVec(0) < 0.999999 {
		t.Fatalf("unexpected outward normal: %v", interaction.GeometricNormal.RawVector().Data)
	}
	if math.Abs(interaction.UV[0]) > 1e-5 || math.Abs(interaction.UV[1]) > 1e-5 {
		t.Fatalf("unexpected UV at axial hit: %v", interaction.UV)
	}
}

func TestKleinBottle4DIntersectRangeRejectsWrongDimensionAndBoxMiss(t *testing.T) {
	k := NewKleinBottle4D(mat.NewVecDense(4, []float64{0, 0, 0, 0}), 1.5, 0.5, 0.06)

	if _, ok := k.IntersectRange(
		mat.NewVecDense(3, []float64{0, 0, 0}),
		mat.NewVecDense(3, []float64{1, 0, 0}),
		0,
		1,
	); ok {
		t.Fatal("expected wrong-dimension ray to miss")
	}

	if _, ok := k.IntersectRange(
		mat.NewVecDense(4, []float64{0, 0, 0, 10}),
		mat.NewVecDense(4, []float64{1, 0, 0, 0}),
		0,
		10,
	); ok {
		t.Fatal("expected ray outside bounding box to miss")
	}
}

func TestKleinBottle4DNormalMatchesFiniteDifference(t *testing.T) {
	k := NewKleinBottle4D(mat.NewVecDense(4, []float64{0, 0, 0, 0}), 1.5, 0.5, 0.06)
	point := mat.NewVecDense(4, []float64{2 + k.Thickness, 0, 0, 0})

	analytic := k.GetNormalVector(point, nil)
	finite := k.GetNormalVectorFiniteDifference(point, nil)

	if mat.Dot(analytic, finite) < 0.999999 {
		t.Fatalf("analytic normal %v does not match finite difference %v", analytic.RawVector().Data, finite.RawVector().Data)
	}
}

func TestKleinBottle4DBuildBoundingBoxUsesCenterAndFourDimensions(t *testing.T) {
	oldDim := utils.Dimension
	utils.SetDimension(3)
	t.Cleanup(func() { utils.SetDimension(oldDim) })

	k := NewKleinBottle4D(mat.NewVecDense(4, []float64{1, -2, 3, -4}), 1.5, 0.5, 0.06)

	pmin, pmax := k.BuildBoundingBox()
	if pmin.Len() != 4 || pmax.Len() != 4 {
		t.Fatalf("expected 4D bounding box, got min=%d max=%d", pmin.Len(), pmax.Len())
	}

	pad := k.Thickness + k.marchEps
	wantMin := []float64{1 - (2 + pad), -2 - (2 + pad), 3 - (0.5 + pad), -4 - (0.5 + pad)}
	wantMax := []float64{1 + (2 + pad), -2 + (2 + pad), 3 + (0.5 + pad), -4 + (0.5 + pad)}

	for i := 0; i < 4; i++ {
		if math.Abs(pmin.AtVec(i)-wantMin[i]) > 1e-12 || math.Abs(pmax.AtVec(i)-wantMax[i]) > 1e-12 {
			t.Fatalf("unexpected bbox axis %d: got [%f, %f] want [%f, %f]", i, pmin.AtVec(i), pmax.AtVec(i), wantMin[i], wantMax[i])
		}
	}
}

func TestKleinBottle4DConstructorRejectsInvalidRadii(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected invalid major/minor radii to panic")
		}
	}()

	NewKleinBottle4D(mat.NewVecDense(4, []float64{0, 0, 0, 0}), 0.5, 0.5, 0.06)
}
