package factory

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"gonum.org/v1/gonum/mat"
)

const (
	ShapeCuboid             = "cuboid"
	ShapeHypercuboid        = "hypercuboid"
	ShapeSphere             = "sphere"
	ShapeHypersphere        = "hypersphere"
	ShapeCircle             = "circle"
	ShapeCylinder           = "cylinder"
	ShapeFiniteCylinder     = "finite cylinder"
	ShapeTriangle           = "triangle"
	ShapePlane              = "plane"
	ShapePolynomial         = "polynomial"
	ShapeImplicitEquation   = "implicit equation"
	ShapeParametricEquation = "parametric equation"
	ShapeParametricCurve    = "parametric curve"
	ShapeKleinBottle        = "klein_bottle"
	ShapeSTL                = "stl"
)

// ParseObjectSpecInSpace dispatches the already-decoded discriminated union.
// Each numerical builder receives its concrete spec, never a string-keyed map.
func ParseObjectSpecInSpace(spec parser.ObjectSpec, space geometry.SceneSpace) ([]shape.Shape, error) {
	dimension := geometry.NewSceneSpace(space.Geometry, space.Dimension).Dimension
	switch definition := spec.Definition.(type) {
	case *parser.CuboidSpec:
		return parseCuboid(definition, spec.Bounds, dimension)
	case *parser.SphereSpec:
		return parseSphere(definition, spec.Bounds, dimension)
	case *parser.CircleSpec:
		return parseCircle(definition, spec.Bounds, dimension)
	case *parser.FiniteCylinderSpec:
		return parseFiniteCylinder(definition, spec.Bounds, dimension)
	case *parser.TriangleSpec:
		return parseTriangle(definition, spec.Bounds, dimension)
	case *parser.PlaneSpec:
		return nil, fmt.Errorf("shape %q is declared but not implemented", spec.Shape)
	case *parser.PolynomialSpec:
		return parsePolynomial(definition, spec.Bounds, dimension)
	case *parser.ImplicitEquationSpec:
		return parseImplicitEquation(definition, spec.Bounds, dimension)
	case *parser.ParametricEquationSpec:
		return parseParametricEquation(definition, spec.Bounds, dimension)
	case *parser.ParametricCurveSpec:
		return parseParametricCurve(definition, spec.Bounds, dimension)
	case *parser.KleinBottleSpec:
		return parseKleinBottle4D(definition, spec.Bounds, dimension)
	case *parser.STLSpec:
		shapes, err := parseShapeForSTLInSpace(definition, space)
		if err != nil {
			return nil, err
		}
		return wrapShapesWithBounds(shapes, spec.Bounds, dimension)
	default:
		return nil, fmt.Errorf("unsupported shape %q", spec.Shape)
	}
}

func parseCuboid(spec *parser.CuboidSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	pmin, err := requiredVector("pmin", spec.PMin, dimension)
	if err != nil {
		return nil, err
	}
	pmax, err := requiredVector("pmax", spec.PMax, dimension)
	if err != nil {
		return nil, err
	}
	cuboid := shape.NewCuboid(mat.NewVecDense(len(pmin), pmin), mat.NewVecDense(len(pmax), pmax))
	return wrapSingleShapeWithBounds(cuboid, bounds, dimension)
}

func parseSphere(spec *parser.SphereSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	center, err := requiredVec("center", spec.Center, dimension)
	if err != nil {
		return nil, err
	}

	radius, err := requiredPositive("r", spec.R)
	if err != nil {
		return nil, err
	}

	sphere := shape.NewSphere(center, radius)
	return wrapSingleShapeWithBounds(sphere, bounds, dimension)
}

func parseCircle(spec *parser.CircleSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	centerValues := spec.Center
	if centerValues == nil {
		centerValues = spec.Position
	}
	center, err := requiredVec("center", centerValues, dimension)
	if err != nil {
		return nil, err
	}

	normal, err := requiredNonZeroVec("normal", spec.Normal, dimension)
	if err != nil {
		return nil, err
	}

	radius, err := requiredPositive("r", spec.R)
	if err != nil {
		return nil, err
	}

	circle := shape.NewCircle(center, normal, radius)
	return wrapSingleShapeWithBounds(circle, bounds, dimension)
}

func parseFiniteCylinder(spec *parser.FiniteCylinderSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	centerValues := spec.Center
	if centerValues == nil {
		centerValues = spec.Position
	}
	center, err := requiredVec("center", centerValues, dimension)
	if err != nil {
		return nil, err
	}

	axis, err := requiredNonZeroVec("axis", spec.Axis, dimension)
	if err != nil {
		return nil, err
	}

	radius, err := requiredPositive("r", spec.R)
	if err != nil {
		return nil, err
	}

	height, err := requiredPositive("height", spec.Height)
	if err != nil {
		return nil, err
	}

	cylinder := shape.NewFiniteCylinder(center, axis, radius, height)
	return wrapSingleShapeWithBounds(cylinder, bounds, dimension)
}

func parseKleinBottle4D(spec *parser.KleinBottleSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	if dimension != 4 {
		return nil, fmt.Errorf("shape %q requires scene dimension 4, got %d", ShapeKleinBottle, dimension)
	}

	center, err := requiredVec("center", spec.Center, dimension)
	if err != nil {
		return nil, err
	}

	majorR, err := requiredPositive("r_major", spec.RMajor)
	if err != nil {
		return nil, err
	}

	minorR, err := requiredPositive("r_minor", spec.RMinor)
	if err != nil {
		return nil, err
	}

	thickness, err := requiredPositive("thickness", spec.Thickness)
	if err != nil {
		return nil, err
	}

	if majorR <= minorR {
		return nil, fmt.Errorf("shape %q requires r_major > r_minor", ShapeKleinBottle)
	}

	klein := shape.NewKleinBottle4D(center, majorR, minorR, thickness)
	return wrapSingleShapeWithBounds(klein, bounds, dimension)
}

func parseTriangle(spec *parser.TriangleSpec, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	p1, err := requiredVec("p1", spec.P1, dimension)
	if err != nil {
		return nil, err
	}

	p2, err := requiredVec("p2", spec.P2, dimension)
	if err != nil {
		return nil, err
	}

	p3, err := requiredVec("p3", spec.P3, dimension)
	if err != nil {
		return nil, err
	}

	triangle := shape.NewTriangle(p1, p2, p3)
	return wrapSingleShapeWithBounds(triangle, bounds, dimension)
}

func wrapSingleShapeWithBounds(s shape.Shape, bounds *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	return wrapShapesWithBounds([]shape.Shape{s}, bounds, dimension)
}

func wrapShapesWithBounds(shapes []shape.Shape, spec *parser.BoundsSpec, dimension int) ([]shape.Shape, error) {
	bounds, ok, err := parseShapeBounds(spec, dimension)
	if err != nil || !ok {
		return shapes, err
	}

	wrapped := make([]shape.Shape, len(shapes))
	for i, inner := range shapes {
		wrapped[i] = shape.NewBoundedShape(inner, bounds)
	}
	return wrapped, nil
}

func parseShapeBounds(spec *parser.BoundsSpec, dimension int) (*shape.Cuboid, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	pmin, err := requiredVector("pmin", spec.PMin, dimension)
	if err != nil {
		return nil, true, fmt.Errorf("bounds requires pmin+pmax: %w", err)
	}
	pmax, err := requiredVector("pmax", spec.PMax, dimension)
	if err != nil {
		return nil, true, fmt.Errorf("bounds requires pmin+pmax: %w", err)
	}
	if err := validateBoundsMinMax(pmin, pmax); err != nil {
		return nil, true, err
	}
	return shape.NewCuboid(
		mat.NewVecDense(len(pmin), pmin),
		mat.NewVecDense(len(pmax), pmax),
	), true, nil
}

func validateBoundsMinMax(pmin, pmax []float64) error {
	for i := range pmin {
		if pmin[i] >= pmax[i] {
			return fmt.Errorf("bounds pmin index %d must be < pmax", i)
		}
	}
	return nil
}

func requiredVector(name string, values []float64, lengths ...int) ([]float64, error) {
	if values == nil {
		return nil, fmt.Errorf("missing required field %q", name)
	}
	validLength := len(lengths) == 0
	for _, length := range lengths {
		if len(values) == length {
			validLength = true
			break
		}
	}
	if !validLength {
		return nil, fmt.Errorf("field %q: expected length %v, got %d", name, lengths, len(values))
	}
	for index, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, fmt.Errorf("field %q index %d must be finite", name, index)
		}
	}
	return values, nil
}

func requiredVec(name string, values []float64, dimension int) (*mat.VecDense, error) {
	values, err := requiredVector(name, values, dimension)
	if err != nil {
		return nil, err
	}
	return mat.NewVecDense(len(values), values), nil
}

func requiredNonZeroVec(name string, values []float64, dimension int) (*mat.VecDense, error) {
	vector, err := requiredVec(name, values, dimension)
	if err != nil {
		return nil, err
	}
	if mat.Norm(vector, 2) == 0 {
		return nil, fmt.Errorf("field %q must be non-zero", name)
	}
	return vector, nil
}

func requiredNumber(name string, value *float64) (float64, error) {
	if value == nil {
		return 0, fmt.Errorf("missing required field %q", name)
	}
	if math.IsNaN(*value) || math.IsInf(*value, 0) {
		return 0, fmt.Errorf("field %q must be finite", name)
	}
	return *value, nil
}

func requiredPositive(name string, value *float64) (float64, error) {
	result, err := requiredNumber(name, value)
	if err != nil {
		return 0, err
	}
	if result <= 0 {
		return 0, fmt.Errorf("field %q must be > 0", name)
	}
	return result, nil
}

func parseShapeForSTLInSpace(spec *parser.STLSpec, space geometry.SceneSpace) ([]shape.Shape, error) {
	dimension := geometry.NewSceneSpace(space.Geometry, space.Dimension).Dimension
	if spec.File == "" {
		return nil, fmt.Errorf("missing required field %q", "file")
	}
	filePath := spec.File
	center, err := requiredVector("center", spec.Center, dimension)
	if err != nil {
		return nil, err
	}
	zDir, err := requiredVector("z_dir", spec.ZDir, dimension)
	if err != nil {
		return nil, err
	}
	xDir, err := requiredVector("x_dir", spec.XDir, dimension)
	if err != nil {
		return nil, err
	}
	scale, err := requiredVector("scale", spec.Scale, dimension)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open STL file %q: %w", filePath, err)
	}
	defer file.Close()

	positionVec := mat.NewVecDense(len(center), center)
	zDirVec := maths.Normalize(mat.NewVecDense(len(zDir), zDir))
	xDirVec := maths.Normalize(mat.NewVecDense(len(xDir), xDir))
	scaleVec := mat.NewVecDense(len(scale), scale)

	transformMatrix := mat.NewDense(4, 4, []float64{
		1, 0, 0, positionVec.AtVec(0),
		0, 1, 0, positionVec.AtVec(1),
		0, 0, 1, positionVec.AtVec(2),
		0, 0, 0, 1,
	})

	yDir := maths.Normalize(maths.Cross2(zDirVec, xDirVec))

	for i := 0; i < 3; i++ {
		transformMatrix.Set(i, 0, xDirVec.AtVec(i))
		transformMatrix.Set(i, 1, yDir.AtVec(i))
		transformMatrix.Set(i, 2, zDirVec.AtVec(i))
		for j := 0; j < 3; j++ {
			transformMatrix.Set(i, j, transformMatrix.At(i, j)*scaleVec.AtVec(j))
		}
	}

	var triangles []shape.Shape

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read STL file %q: %w", filePath, err)
		}
		return nil, fmt.Errorf("STL file %q is empty", filePath)
	}
	firstLine := scanner.Text()
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("seek STL file %q: %w", filePath, err)
	}

	if strings.HasPrefix(firstLine, "solid") {
		scanner := bufio.NewScanner(file)
		var p1, p2, p3 *mat.VecDense

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(line, "vertex") {
				continue
			}

			var x, y, z float64
			if _, err := fmt.Sscanf(line, "vertex %f %f %f", &x, &y, &z); err != nil {
				return nil, fmt.Errorf("parse STL vertex %q: %w", line, err)
			}

			if p1 == nil {
				p1 = mat.NewVecDense(3, []float64{x, y, z})
			} else if p2 == nil {
				p2 = mat.NewVecDense(3, []float64{x, y, z})
			} else if p3 == nil {
				p3 = mat.NewVecDense(3, []float64{x, y, z})
			}

			if p3 != nil {
				triangle := shape.NewTriangle(
					transformVertexWithMatrix(p1, transformMatrix),
					transformVertexWithMatrix(p2, transformMatrix),
					transformVertexWithMatrix(p3, transformMatrix),
				)
				triangles = append(triangles, triangle)
				p1, p2, p3 = nil, nil, nil
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scan STL file %q: %w", filePath, err)
		}
	} else {
		header := make([]byte, 80)
		if _, err := file.Read(header); err != nil {
			return nil, fmt.Errorf("read STL header %q: %w", filePath, err)
		}

		var numTriangles uint32
		if err := binary.Read(file, binary.LittleEndian, &numTriangles); err != nil {
			return nil, fmt.Errorf("read STL triangle count %q: %w", filePath, err)
		}

		for i := uint32(0); i < numTriangles; i++ {
			normal := make([]byte, 12)
			if _, err := file.Read(normal); err != nil {
				return nil, fmt.Errorf("read STL normal %q triangle %d: %w", filePath, i, err)
			}

			var vertices [9]float32
			if err := binary.Read(file, binary.LittleEndian, &vertices); err != nil {
				return nil, fmt.Errorf("read STL vertices %q triangle %d: %w", filePath, i, err)
			}

			var attrByteCount uint16
			if err := binary.Read(file, binary.LittleEndian, &attrByteCount); err != nil {
				return nil, fmt.Errorf("read STL attribute count %q triangle %d: %w", filePath, i, err)
			}

			p1 := mat.NewVecDense(3, []float64{float64(vertices[0]), float64(vertices[1]), float64(vertices[2])})
			p2 := mat.NewVecDense(3, []float64{float64(vertices[3]), float64(vertices[4]), float64(vertices[5])})
			p3 := mat.NewVecDense(3, []float64{float64(vertices[6]), float64(vertices[7]), float64(vertices[8])})
			triangle := shape.NewTriangle(
				transformVertexWithMatrix(p1, transformMatrix),
				transformVertexWithMatrix(p2, transformMatrix),
				transformVertexWithMatrix(p3, transformMatrix),
			)
			triangles = append(triangles, triangle)
		}
	}

	if len(triangles) == 0 {
		return nil, fmt.Errorf("STL file %q produced no triangles", filePath)
	}

	return triangles, nil
}

func transformVertexWithMatrix(vertex *mat.VecDense, transformMatrix *mat.Dense) *mat.VecDense {
	vertexHomogeneous := mat.NewVecDense(vertex.Len()+1, []float64{
		vertex.AtVec(0),
		vertex.AtVec(1),
		vertex.AtVec(2),
		1.0,
	})

	transformed := new(mat.VecDense)
	transformed.MulVec(transformMatrix, vertexHomogeneous)

	return mat.NewVecDense(vertex.Len(), []float64{
		transformed.AtVec(0),
		transformed.AtVec(1),
		transformed.AtVec(2),
	})
}
