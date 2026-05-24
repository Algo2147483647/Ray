package maths

import (
	"errors"
	"math"
	"math/cmplx"
)

const (
	DefaultTol     = 1e-12
	DefaultMaxIter = 10000
)

var (
	ErrNoSolution        = errors.New("no solution")
	ErrInfiniteSolutions = errors.New("infinite solutions")
	ErrInvalidInput      = errors.New("invalid input")
	ErrNonSquareSystem   = errors.New("linear system must be square")
	ErrSingularMatrix    = errors.New("singular matrix")
	ErrNotConverged      = errors.New("not converged")
	ErrDegreeTooHigh     = errors.New("polynomial degree too high")
	ErrZeroPolynomial    = errors.New("zero polynomial")
)

func isZero(x float64) bool {
	return math.Abs(x) <= DefaultTol
}

func isZeroComplex(z complex128) bool {
	return cmplx.Abs(z) <= DefaultTol
}
