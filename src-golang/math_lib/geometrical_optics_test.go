package math_lib

import "testing"

func TestCauchyDispersion(t *testing.T) {
	A := 1.500
	B := 50000.0
	C := 0.0
	t.Log(CauchyDispersion(WavelengthMax, A, B, C))
	t.Log(CauchyDispersion(WavelengthMin, A, B, C))
}
