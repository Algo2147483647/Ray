package factory

import (
	"fmt"

	"github.com/Algo2147483647/ray/engine/maths/exprdiff"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"github.com/expr-lang/expr/vm"
	"gonum.org/v1/gonum/mat"
)

type parametricExprSurface struct {
	xProgram *vm.Program
	yProgram *vm.Program
	zProgram *vm.Program
	du       [3]*vm.Program
	dv       [3]*vm.Program
	Mem      *exprEnvPool
}

func parseParametricExprSurface(surfaceDef map[string]interface{}) (shape.ParametricFunction, shape.ParametricDerivative, error) {
	constants, err := parseParametricExprConstants(surfaceDef)
	if err != nil {
		return nil, nil, err
	}

	xSource, err := utils.RequiredStringField(surfaceDef, "x")
	if err != nil {
		return nil, nil, err
	}
	ySource, err := utils.RequiredStringField(surfaceDef, "y")
	if err != nil {
		return nil, nil, err
	}
	zSource, err := utils.RequiredStringField(surfaceDef, "z")
	if err != nil {
		return nil, nil, err
	}

	surface := &parametricExprSurface{
		Mem: newExprEnvPool(constants, "u", "v"),
	}
	if surface.xProgram, err = compileParametricExprProgram("surface.x", xSource, constants); err != nil {
		return nil, nil, err
	}
	if surface.yProgram, err = compileParametricExprProgram("surface.y", ySource, constants); err != nil {
		return nil, nil, err
	}
	if surface.zProgram, err = compileParametricExprProgram("surface.z", zSource, constants); err != nil {
		return nil, nil, err
	}

	if derivativeDef, ok, err := utils.OptionalMapField(surfaceDef, "derivative"); err != nil {
		return nil, nil, err
	} else if ok {
		if err := surface.compileDerivative(derivativeDef, constants); err != nil {
			return nil, nil, err
		}
	} else if err := surface.compileAutoDerivative([3]string{xSource, ySource, zSource}, constants); err != nil {
		return nil, nil, err
	}

	if surface.hasDerivative() {
		return surface.evaluate, surface.derivative, nil
	}
	return surface.evaluate, nil, nil
}

func parseParametricExprConstants(surfaceDef map[string]interface{}) (map[string]float64, error) {
	constantsDef, ok, err := utils.OptionalMapField(surfaceDef, "constants")
	if err != nil || !ok {
		return nil, err
	}

	reserved := implicitExprBaseEnv()
	reserved["u"] = 0.0
	reserved["v"] = 0.0

	constants := make(map[string]float64, len(constantsDef))
	for name := range constantsDef {
		if _, ok := reserved[name]; ok {
			return nil, fmt.Errorf("constants field %q is reserved", name)
		}
		value, err := utils.RequiredFloat64Field(constantsDef, name)
		if err != nil {
			return nil, fmt.Errorf("constants.%s: %w", name, err)
		}
		constants[name] = value
	}
	return constants, nil
}

func (s *parametricExprSurface) compileDerivative(derivativeDef map[string]interface{}, constants map[string]float64) error {
	duDef, ok, err := utils.OptionalMapField(derivativeDef, "du")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf(`derivative missing required field "du"`)
	}
	dvDef, ok, err := utils.OptionalMapField(derivativeDef, "dv")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf(`derivative missing required field "dv"`)
	}

	for axis, name := range []string{"x", "y", "z"} {
		source, err := utils.RequiredStringField(duDef, name)
		if err != nil {
			return fmt.Errorf("derivative.du.%s: %w", name, err)
		}
		program, err := compileParametricExprProgram("derivative.du."+name, source, constants)
		if err != nil {
			return err
		}
		s.du[axis] = program

		source, err = utils.RequiredStringField(dvDef, name)
		if err != nil {
			return fmt.Errorf("derivative.dv.%s: %w", name, err)
		}
		program, err = compileParametricExprProgram("derivative.dv."+name, source, constants)
		if err != nil {
			return err
		}
		s.dv[axis] = program
	}
	return nil
}

func (s *parametricExprSurface) compileAutoDerivative(sources [3]string, constants map[string]float64) error {
	for axis, source := range sources {
		derivatives, ok := exprdiff.Derivatives(source, "u", "v")
		if !ok {
			return nil
		}
		var err error
		s.du[axis], err = compileParametricExprProgram(fmt.Sprintf("derivative.du.%d", axis), derivatives[0], constants)
		if err != nil {
			return err
		}
		s.dv[axis], err = compileParametricExprProgram(fmt.Sprintf("derivative.dv.%d", axis), derivatives[1], constants)
		if err != nil {
			return err
		}
	}
	return nil
}

func compileParametricExprProgram(label, source string, constants map[string]float64) (*vm.Program, error) {
	return compileExprProgram(label, source, constants, "u", "v")
}

func (s *parametricExprSurface) evaluate(u, v float64) *mat.VecDense {
	if s == nil {
		return nil
	}
	env := s.Mem.get(u, v)
	x := runImplicitExprProgram(s.xProgram, env)
	y := runImplicitExprProgram(s.yProgram, env)
	z := runImplicitExprProgram(s.zProgram, env)
	s.Mem.put(env)
	if !implicitExprIsFinite(x) || !implicitExprIsFinite(y) || !implicitExprIsFinite(z) {
		return nil
	}
	return mat.NewVecDense(3, []float64{x, y, z})
}

func (s *parametricExprSurface) derivative(u, v float64, du, dv *mat.VecDense) (*mat.VecDense, *mat.VecDense) {
	if s == nil || !s.hasDerivative() {
		return nil, nil
	}
	if du == nil || du.Len() != 3 {
		du = mat.NewVecDense(3, nil)
	} else {
		du.Zero()
	}
	if dv == nil || dv.Len() != 3 {
		dv = mat.NewVecDense(3, nil)
	} else {
		dv.Zero()
	}

	env := s.Mem.get(u, v)
	for axis := 0; axis < 3; axis++ {
		duValue := runImplicitExprProgram(s.du[axis], env)
		dvValue := runImplicitExprProgram(s.dv[axis], env)
		if !implicitExprIsFinite(duValue) || !implicitExprIsFinite(dvValue) {
			s.Mem.put(env)
			return nil, nil
		}
		du.SetVec(axis, duValue)
		dv.SetVec(axis, dvValue)
	}
	s.Mem.put(env)
	return du, dv
}

func (s *parametricExprSurface) hasDerivative() bool {
	for axis := 0; axis < 3; axis++ {
		if s.du[axis] == nil || s.dv[axis] == nil {
			return false
		}
	}
	return true
}
