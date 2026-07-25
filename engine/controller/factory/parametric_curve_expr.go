package factory

import (
	"fmt"
	"math"
	"sync"

	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"github.com/expr-lang/expr/vm"
	"gonum.org/v1/gonum/mat"
)

type parametricExprCurve struct {
	xProgram       *vm.Program
	yProgram       *vm.Program
	zProgram       *vm.Program
	radiusProgram  *vm.Program
	constRadius    float64
	hasConstRadius bool
	derivative     [3]*vm.Program
	Mem            parametricCurveExprCalculateStorage
}

type parametricCurveExprCalculateStorage struct {
	baseEnv map[string]interface{}
	envPool sync.Pool
}

func parseParametricExprCurve(curveDef map[string]interface{}) (shape.ParametricCurveFunction, shape.ParametricCurveDerivative, shape.ParametricCurveRadius, error) {
	constants, err := parseParametricCurveExprConstants(curveDef)
	if err != nil {
		return nil, nil, nil, err
	}

	xSource, err := utils.RequiredStringField(curveDef, "x")
	if err != nil {
		return nil, nil, nil, err
	}
	ySource, err := utils.RequiredStringField(curveDef, "y")
	if err != nil {
		return nil, nil, nil, err
	}
	zSource, err := utils.RequiredStringField(curveDef, "z")
	if err != nil {
		return nil, nil, nil, err
	}

	curve := &parametricExprCurve{
		Mem: newParametricCurveExprCalculateStorage(constants),
	}
	if curve.xProgram, err = compileParametricCurveExprProgram("curve.x", xSource, constants); err != nil {
		return nil, nil, nil, err
	}
	if curve.yProgram, err = compileParametricCurveExprProgram("curve.y", ySource, constants); err != nil {
		return nil, nil, nil, err
	}
	if curve.zProgram, err = compileParametricCurveExprProgram("curve.z", zSource, constants); err != nil {
		return nil, nil, nil, err
	}
	if err := curve.compileRadius(curveDef, constants); err != nil {
		return nil, nil, nil, err
	}

	if derivativeDef, ok, err := utils.OptionalMapField(curveDef, "derivative"); err != nil {
		return nil, nil, nil, err
	} else if ok {
		if err := curve.compileDerivative(derivativeDef, constants); err != nil {
			return nil, nil, nil, err
		}
	} else if err := curve.compileAutoDerivative([3]string{xSource, ySource, zSource}, constants); err != nil {
		return nil, nil, nil, err
	}

	if curve.hasDerivative() {
		return curve.evaluate, curve.derivativeAt, curve.radiusAt, nil
	}
	return curve.evaluate, nil, curve.radiusAt, nil
}

func parseParametricCurveExprConstants(curveDef map[string]interface{}) (map[string]float64, error) {
	constantsDef, ok, err := utils.OptionalMapField(curveDef, "constants")
	if err != nil || !ok {
		return nil, err
	}

	reserved := implicitExprBaseEnv()
	reserved["t"] = 0.0

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

func (c *parametricExprCurve) compileRadius(curveDef map[string]interface{}, constants map[string]float64) error {
	raw, ok := curveDef["radius"]
	if !ok {
		raw, ok = curveDef["r"]
	}
	if !ok {
		return fmt.Errorf(`parametric curve expr requires "radius"`)
	}

	switch value := raw.(type) {
	case string:
		program, err := compileParametricCurveExprProgram("curve.radius", value, constants)
		if err != nil {
			return err
		}
		c.radiusProgram = program
		return nil
	default:
		radius, err := utils.RequiredFloat64Field(map[string]interface{}{"radius": raw}, "radius")
		if err != nil {
			return err
		}
		if radius <= 0 {
			return fmt.Errorf("radius must be > 0")
		}
		c.constRadius = radius
		c.hasConstRadius = true
		return nil
	}
}

func (c *parametricExprCurve) compileDerivative(derivativeDef map[string]interface{}, constants map[string]float64) error {
	for axis, name := range []string{"x", "y", "z"} {
		source, err := utils.RequiredStringField(derivativeDef, name)
		if err != nil {
			return fmt.Errorf("derivative.%s: %w", name, err)
		}
		program, err := compileParametricCurveExprProgram("derivative."+name, source, constants)
		if err != nil {
			return err
		}
		c.derivative[axis] = program
	}
	return nil
}

func (c *parametricExprCurve) compileAutoDerivative(sources [3]string, constants map[string]float64) error {
	for axis, source := range sources {
		derivatives, ok := autodiffExpr(source, "t")
		if !ok {
			return nil
		}
		var err error
		c.derivative[axis], err = compileParametricCurveExprProgram(fmt.Sprintf("derivative.%d", axis), derivatives[0], constants)
		if err != nil {
			return err
		}
	}
	return nil
}

func compileParametricCurveExprProgram(label, source string, constants map[string]float64) (*vm.Program, error) {
	return compileExprProgram(label, source, constants, "t")
}

func (c *parametricExprCurve) evaluate(t float64) *mat.VecDense {
	if c == nil {
		return nil
	}
	env := c.Mem.getEnv(t)
	x := runImplicitExprProgram(c.xProgram, env)
	y := runImplicitExprProgram(c.yProgram, env)
	z := runImplicitExprProgram(c.zProgram, env)
	c.Mem.putEnv(env)
	if !implicitExprIsFinite(x) || !implicitExprIsFinite(y) || !implicitExprIsFinite(z) {
		return nil
	}
	return mat.NewVecDense(3, []float64{x, y, z})
}

func (c *parametricExprCurve) radiusAt(t float64) float64 {
	if c == nil {
		return math.NaN()
	}
	if c.hasConstRadius {
		return c.constRadius
	}
	env := c.Mem.getEnv(t)
	radius := runImplicitExprProgram(c.radiusProgram, env)
	c.Mem.putEnv(env)
	return radius
}

func (c *parametricExprCurve) derivativeAt(t float64, res *mat.VecDense) *mat.VecDense {
	if c == nil || !c.hasDerivative() {
		return nil
	}
	if res == nil || res.Len() != 3 {
		res = mat.NewVecDense(3, nil)
	} else {
		res.Zero()
	}

	env := c.Mem.getEnv(t)
	for axis := 0; axis < 3; axis++ {
		value := runImplicitExprProgram(c.derivative[axis], env)
		if !implicitExprIsFinite(value) {
			c.Mem.putEnv(env)
			return nil
		}
		res.SetVec(axis, value)
	}
	c.Mem.putEnv(env)
	return res
}

func (c *parametricExprCurve) hasDerivative() bool {
	for axis := 0; axis < 3; axis++ {
		if c.derivative[axis] == nil {
			return false
		}
	}
	return true
}

func newParametricCurveExprCalculateStorage(constants map[string]float64) parametricCurveExprCalculateStorage {
	baseEnv := implicitExprBaseEnvWithConstants(constants)
	baseEnv["t"] = 0.0
	mem := parametricCurveExprCalculateStorage{
		baseEnv: baseEnv,
	}
	mem.envPool.New = func() interface{} {
		return cloneImplicitExprEnv(baseEnv)
	}
	return mem
}

func (m *parametricCurveExprCalculateStorage) getEnv(t float64) map[string]interface{} {
	if m == nil {
		env := implicitExprBaseEnv()
		env["t"] = t
		return env
	}
	raw := m.envPool.Get()
	env, ok := raw.(map[string]interface{})
	if !ok || env == nil {
		env = cloneImplicitExprEnv(m.baseEnv)
	}
	env["t"] = t
	return env
}

func (m *parametricCurveExprCalculateStorage) putEnv(env map[string]interface{}) {
	if m == nil || env == nil {
		return
	}
	m.envPool.Put(env)
}
