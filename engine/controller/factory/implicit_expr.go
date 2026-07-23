package factory

import (
	"fmt"
	"math"
	"sync"

	"github.com/Algo2147483647/ray/engine/utils"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gonum.org/v1/gonum/mat"
)

type implicitExprField struct {
	program   *vm.Program
	gradientX *vm.Program
	gradientY *vm.Program
	gradientZ *vm.Program
	constants map[string]float64
	Mem       implicitExprCalculateStorage
}

type implicitExprCalculateStorage struct {
	baseEnv map[string]interface{}
	envPool sync.Pool
}

func parseImplicitExprField(
	fieldDef map[string]interface{},
) (
	func(*mat.VecDense) float64,
	func(point, res *mat.VecDense) *mat.VecDense,
	error,
) {
	source, err := utils.RequiredStringField(fieldDef, "expr")
	if err != nil {
		return nil, nil, err
	}

	constants, err := parseImplicitExprConstants(fieldDef)
	if err != nil {
		return nil, nil, err
	}

	program, err := compileImplicitExprProgram("expr", source, constants)
	if err != nil {
		return nil, nil, err
	}

	field := &implicitExprField{
		program:   program,
		constants: constants,
		Mem:       newImplicitExprCalculateStorage(constants),
	}

	if gradientDef, ok, err := utils.OptionalMapField(fieldDef, "gradient"); err != nil {
		return nil, nil, err
	} else if ok {
		if err := field.compileGradient(gradientDef); err != nil {
			return nil, nil, err
		}
	} else if gx, gy, gz, ok := autodiffImplicitExpr(source); ok {
		if err := field.compileGradient(map[string]interface{}{"x": gx, "y": gy, "z": gz}); err != nil {
			return nil, nil, err
		}
	}

	return field.evaluate, field.gradient, nil
}

func parseImplicitExprConstants(fieldDef map[string]interface{}) (map[string]float64, error) {
	constantsDef, ok, err := utils.OptionalMapField(fieldDef, "constants")
	if err != nil || !ok {
		return nil, err
	}

	constants := make(map[string]float64, len(constantsDef))
	for name := range constantsDef {
		if _, reserved := implicitExprBaseEnv()[name]; reserved {
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

func (f *implicitExprField) compileGradient(gradientDef map[string]interface{}) error {
	xSource, err := utils.RequiredStringField(gradientDef, "x")
	if err != nil {
		return fmt.Errorf("gradient.x: %w", err)
	}
	ySource, err := utils.RequiredStringField(gradientDef, "y")
	if err != nil {
		return fmt.Errorf("gradient.y: %w", err)
	}
	zSource, err := utils.RequiredStringField(gradientDef, "z")
	if err != nil {
		return fmt.Errorf("gradient.z: %w", err)
	}

	if f.gradientX, err = compileImplicitExprProgram("gradient.x", xSource, f.constants); err != nil {
		return err
	}
	if f.gradientY, err = compileImplicitExprProgram("gradient.y", ySource, f.constants); err != nil {
		return err
	}
	if f.gradientZ, err = compileImplicitExprProgram("gradient.z", zSource, f.constants); err != nil {
		return err
	}
	return nil
}

func compileImplicitExprProgram(label, source string, constants map[string]float64) (*vm.Program, error) {
	env := implicitExprBaseEnvWithConstants(constants)
	program, err := expr.Compile(source, expr.Env(env), expr.AsFloat64())
	if err != nil {
		return nil, fmt.Errorf("%s: compile expression: %w", label, err)
	}
	return program, nil
}

func (f *implicitExprField) evaluate(point *mat.VecDense) float64 {
	if f == nil || point == nil || point.Len() < 3 {
		return math.NaN()
	}
	x, y, z := point.AtVec(0), point.AtVec(1), point.AtVec(2)
	env := f.Mem.getEnv(x, y, z)
	value := runImplicitExprProgram(f.program, env)
	f.Mem.putEnv(env)
	return value
}

func (f *implicitExprField) gradient(point, res *mat.VecDense) *mat.VecDense {
	if f == nil || f.gradientX == nil || f.gradientY == nil || f.gradientZ == nil || point == nil {
		return nil
	}
	if res == nil || res.Len() != point.Len() {
		res = mat.NewVecDense(point.Len(), nil)
	} else {
		res.Zero()
	}

	x, y, z := point.AtVec(0), point.AtVec(1), point.AtVec(2)
	env := f.Mem.getEnv(x, y, z)
	gx := runImplicitExprProgram(f.gradientX, env)
	gy := runImplicitExprProgram(f.gradientY, env)
	gz := runImplicitExprProgram(f.gradientZ, env)
	f.Mem.putEnv(env)
	if !implicitExprIsFinite(gx) || !implicitExprIsFinite(gy) || !implicitExprIsFinite(gz) {
		return nil
	}

	res.SetVec(0, gx)
	res.SetVec(1, gy)
	res.SetVec(2, gz)
	return res
}

func runImplicitExprProgram(program *vm.Program, env map[string]interface{}) float64 {
	output, err := expr.Run(program, env)
	if err != nil {
		return math.NaN()
	}
	value, ok := output.(float64)
	if !ok || !implicitExprIsFinite(value) {
		return math.NaN()
	}
	return value
}

func newImplicitExprCalculateStorage(constants map[string]float64) implicitExprCalculateStorage {
	baseEnv := implicitExprBaseEnvWithConstants(constants)
	mem := implicitExprCalculateStorage{
		baseEnv: baseEnv,
	}
	mem.envPool.New = func() interface{} {
		return cloneImplicitExprEnv(baseEnv)
	}
	return mem
}

func (m *implicitExprCalculateStorage) getEnv(x, y, z float64) map[string]interface{} {
	if m == nil {
		env := implicitExprBaseEnv()
		env["x"] = x
		env["y"] = y
		env["z"] = z
		return env
	}
	raw := m.envPool.Get()
	env, ok := raw.(map[string]interface{})
	if !ok || env == nil {
		env = cloneImplicitExprEnv(m.baseEnv)
	}
	env["x"] = x
	env["y"] = y
	env["z"] = z
	return env
}

func (m *implicitExprCalculateStorage) putEnv(env map[string]interface{}) {
	if m == nil || env == nil {
		return
	}
	m.envPool.Put(env)
}

func implicitExprBaseEnvWithConstants(constants map[string]float64) map[string]interface{} {
	env := implicitExprBaseEnv()
	for name, value := range constants {
		env[name] = value
	}
	return env
}

func cloneImplicitExprEnv(source map[string]interface{}) map[string]interface{} {
	env := make(map[string]interface{}, len(source))
	for key, value := range source {
		env[key] = value
	}
	return env
}

func implicitExprBaseEnv() map[string]interface{} {
	return map[string]interface{}{
		"x":  0.0,
		"y":  0.0,
		"z":  0.0,
		"pi": math.Pi,
		"e":  math.E,

		"abs":   math.Abs,
		"sqrt":  math.Sqrt,
		"sin":   math.Sin,
		"cos":   math.Cos,
		"tan":   math.Tan,
		"asin":  math.Asin,
		"acos":  math.Acos,
		"atan":  math.Atan,
		"atan2": math.Atan2,
		"sinh":  math.Sinh,
		"cosh":  math.Cosh,
		"tanh":  math.Tanh,
		"exp":   math.Exp,
		"log":   math.Log,
		"log10": math.Log10,
		"floor": math.Floor,
		"ceil":  math.Ceil,
		"round": math.Round,
		"pow":   math.Pow,
		"min":   math.Min,
		"max":   math.Max,
		"clamp": clampFloat64,
		"sign":  signFloat64,
	}
}

func clampFloat64(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func signFloat64(value float64) float64 {
	switch {
	case value < 0:
		return -1
	case value > 0:
		return 1
	default:
		return 0
	}
}

func implicitExprIsFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
