package utils

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"testing"
)

func TestDiffuseReflectStaysInNormalHemisphere(t *testing.T) {
	normal := mat.NewVecDense(3, []float64{0, 0, 1})
	incidentA := mat.NewVecDense(3, []float64{0, 0, -1})
	incidentB := mat.NewVecDense(3, []float64{1, 0, -1})

	for i := 0; i < 200; i++ {
		dirA := DiffuseReflect(incidentA, normal)
		dirB := DiffuseReflect(incidentB, normal)

		if dot := mat.Dot(dirA, normal); dot < -1e-9 {
			t.Fatalf("sample A escaped hemisphere: dot=%v", dot)
		}
		if dot := mat.Dot(dirB, normal); dot < -1e-9 {
			t.Fatalf("sample B escaped hemisphere: dot=%v", dot)
		}
		if norm := mat.Norm(dirA, 2); math.Abs(norm-1) > 1e-9 {
			t.Fatalf("sample A is not normalized: norm=%v", norm)
		}
		if norm := mat.Norm(dirB, 2); math.Abs(norm-1) > 1e-9 {
			t.Fatalf("sample B is not normalized: norm=%v", norm)
		}
	}
}

func TestFresnelSchlickMatchesExpectedEndpoints(t *testing.T) {
	normalIncidence := FresnelSchlick(1, 1.0, 1.5)
	if math.Abs(normalIncidence-0.04) > 1e-9 {
		t.Fatalf("unexpected normal-incidence reflectance: got %v want 0.04", normalIncidence)
	}

	grazingIncidence := FresnelSchlick(0, 1.0, 1.5)
	if math.Abs(grazingIncidence-1.0) > 1e-9 {
		t.Fatalf("unexpected grazing reflectance: got %v want 1", grazingIncidence)
	}
}

func TestHasTotalInternalReflection(t *testing.T) {
	normal := mat.NewVecDense(3, []float64{0, 0, 1})
	incident := mat.NewVecDense(3, []float64{math.Sqrt(3) / 2, 0, -0.5})

	if !HasTotalInternalReflection(incident, normal, 1.5) {
		t.Fatal("expected total internal reflection for steep inside-to-air ray")
	}
}

func TestDiffuseReflectDoesNotReturnPooledVector(t *testing.T) {
	normal := mat.NewVecDense(3, []float64{0, 0, 1})
	incident := mat.NewVecDense(3, []float64{0, 0, -1})

	first := DiffuseReflect(incident, normal)
	firstSnapshot := append([]float64(nil), first.RawVector().Data...)

	for i := 0; i < 32; i++ {
		_ = DiffuseReflect(incident, normal)
	}

	for i, want := range firstSnapshot {
		if got := first.AtVec(i); math.Abs(got-want) > 1e-12 {
			t.Fatalf("sample was mutated after later calls at index %d: got %v want %v", i, got, want)
		}
	}
}
