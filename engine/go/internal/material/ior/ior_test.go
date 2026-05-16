package ior

import (
	"math"
	"testing"
)

func TestConstantIOR(t *testing.T) {
	model := NewConstant(1.5)
	if model.IsDispersive() {
		t.Fatal("constant IOR should not be dispersive")
	}
	if got := model.Evaluate(430); got != 1.5 {
		t.Fatalf("unexpected constant eta: got %f", got)
	}
}

func TestCauchyUsesMicrometerWavelength(t *testing.T) {
	model := NewCauchy(1.5, 0.004, 0.0001)
	got := model.Evaluate(500)
	want := 1.5 + 0.004/(0.5*0.5) + 0.0001/(0.5*0.5*0.5*0.5)
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("unexpected cauchy eta: got %f want %f", got, want)
	}
	if !model.IsDispersive() {
		t.Fatal("cauchy model with non-zero B/C should be dispersive")
	}
}

func TestIsValidEta(t *testing.T) {
	if !IsValidEta(1.000293) {
		t.Fatal("expected positive finite eta to be valid")
	}
	if IsValidEta(0) || IsValidEta(math.Inf(1)) || IsValidEta(math.NaN()) {
		t.Fatal("expected non-positive and non-finite eta values to be invalid")
	}
}
