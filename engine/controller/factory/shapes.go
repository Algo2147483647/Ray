package factory

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/Algo2147483647/ray/engine/utils/example_lib"
	"math"
	"os"
	"strings"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/utils"
	"gonum.org/v1/gonum/mat"
)

const (
	ShapeCuboid            = "cuboid"
	ShapeHypercube         = "hypercube"
	ShapeHypercuboid       = "hypercuboid"
	ShapeSphere            = "sphere"
	ShapeHypersphere       = "hypersphere"
	ShapeCircle            = "circle"
	ShapeCylinder          = "cylinder"
	ShapeFiniteCylinder    = "finite cylinder"
	ShapeTriangle          = "triangle"
	ShapePlane             = "plane"
	ShapeQuadraticEquation = "quadratic equation"
	ShapeCubicEquation     = "cubic equation"
	ShapeFourOrderEquation = "four-order equation"
	ShapeImplicitEquation  = "implicit equation"
	ShapePolynomialSurface = "polynomial surface"
	ShapeKleinBottle       = "klein_bottle"
	ShapeSTL               = "stl"
)

func ParseShape(objDef map[string]interface{}) ([]shape.Shape, error) {
	shapeName, err := utils.RequiredStringField(objDef, "shape")
	if err != nil {
		return nil, err
	}

	switch shapeName {
	case ShapeCuboid, ShapeHypercuboid:
		return parseCuboid(objDef)

	case ShapeHypercube:
		return parseHypercube(objDef)

	case ShapeSphere, ShapeHypersphere:
		return parseSphere(objDef)

	case ShapeCircle:
		return parseCircle(objDef)

	case ShapeCylinder, ShapeFiniteCylinder:
		return parseFiniteCylinder(objDef)

	case ShapeTriangle:
		return parseTriangle(objDef)

	case ShapePlane:
		return nil, fmt.Errorf("shape %q is declared but not implemented", shapeName)

	case ShapeQuadraticEquation:
		return parseQuadraticEquation(objDef)

	case ShapeCubicEquation:
		return parseCubicEquation(objDef)

	case ShapeFourOrderEquation:
		return parseFourOrderEquation(objDef)

	case ShapeImplicitEquation:
		return parseImplicitEquation(objDef)

	case ShapePolynomialSurface:
		return parsePolynomialSurface(objDef)

	case ShapeKleinBottle:
		return parseKleinBottle4D(objDef)

	case ShapeSTL:
		shapes, err := ParseShapeForSTL(objDef)
		if err != nil {
			return nil, err
		}
		return wrapShapesWithBounds(shapes, objDef)

	default:
		return nil, fmt.Errorf("unsupported shape %q", shapeName)
	}
}

func parseHypercube(objDef map[string]interface{}) ([]shape.Shape, error) {
	shapes, err := parseCuboid(objDef)
	if err != nil {
		return nil, err
	}
	for _, s := range shapes {
		cuboid, ok := s.(*shape.Cuboid)
		if !ok {
			if bounded, boundedOK := s.(*shape.BoundedShape); boundedOK {
				cuboid, ok = bounded.Shape.(*shape.Cuboid)
			}
			continue
		}
		side := cuboid.Pmax.AtVec(0) - cuboid.Pmin.AtVec(0)
		for axis := 1; axis < cuboid.Pmin.Len(); axis++ {
			if diff := cuboid.Pmax.AtVec(axis) - cuboid.Pmin.AtVec(axis); math.Abs(diff-side) > utils.EPS {
				return nil, fmt.Errorf("hypercube requires equal side lengths, axis %d has %g instead of %g", axis, diff, side)
			}
		}
	}
	return shapes, nil
}

func parseCuboid(objDef map[string]interface{}) ([]shape.Shape, error) {
	pmin, err := utils.RequiredFloat64SliceField(objDef, "pmin", utils.Dimension)
	if err != nil {
		return nil, err
	}

	pmax, err := utils.RequiredFloat64SliceField(objDef, "pmax", utils.Dimension)
	if err != nil {
		return nil, err
	}

	cuboid := shape.NewCuboid(utils.NewVec(pmin), utils.NewVec(pmax))
	return finishEngravableShape(cuboid, objDef)
}

func parseSphere(objDef map[string]interface{}) ([]shape.Shape, error) {
	center, err := utils.RequiredVec(objDef, "center", utils.Dimension)
	if err != nil {
		return nil, err
	}

	radius, err := utils.RequiredPositiveFloat(objDef, "r")
	if err != nil {
		return nil, err
	}

	sphere := shape.NewSphere(center, radius)
	return finishEngravableShape(sphere, objDef)
}

func parseCircle(objDef map[string]interface{}) ([]shape.Shape, error) {
	center, err := utils.RequiredVec(objDef, "center", utils.Dimension)
	if err != nil {
		return nil, err
	}

	normal, err := utils.RequiredNonZeroVec(objDef, "normal", utils.Dimension)
	if err != nil {
		return nil, err
	}

	radius, err := utils.RequiredPositiveFloat(objDef, "r")
	if err != nil {
		return nil, err
	}

	circle := shape.NewCircle(center, normal, radius)
	return finishEngravableShape(circle, objDef)
}

func parseFiniteCylinder(objDef map[string]interface{}) ([]shape.Shape, error) {
	center, err := utils.RequiredVec(objDef, "center", utils.Dimension)
	if err != nil {
		return nil, err
	}

	axis, err := utils.RequiredNonZeroVec(objDef, "axis", utils.Dimension)
	if err != nil {
		return nil, err
	}

	radius, err := utils.RequiredPositiveFloat(objDef, "r")
	if err != nil {
		return nil, err
	}

	height, err := utils.RequiredPositiveFloat(objDef, "height")
	if err != nil {
		return nil, err
	}

	cylinder := shape.NewFiniteCylinder(center, axis, radius, height)
	return finishEngravableShape(cylinder, objDef)
}

func parseKleinBottle4D(objDef map[string]interface{}) ([]shape.Shape, error) {
	if utils.Dimension != 4 {
		return nil, fmt.Errorf("shape %q requires render dimension 4, got %d", ShapeKleinBottle, utils.Dimension)
	}

	center, err := utils.RequiredVec(objDef, "center", utils.Dimension)
	if err != nil {
		return nil, err
	}

	majorR, err := utils.RequiredPositiveFloat(objDef, "r_major")
	if err != nil {
		return nil, err
	}

	minorR, err := utils.RequiredPositiveFloat(objDef, "r_minor")
	if err != nil {
		return nil, err
	}

	thickness, err := utils.RequiredPositiveFloat(objDef, "thickness")
	if err != nil {
		return nil, err
	}

	if majorR <= minorR {
		return nil, fmt.Errorf("shape %q requires r_major > r_minor", ShapeKleinBottle)
	}

	klein := shape.NewKleinBottle4D(center, majorR, minorR, thickness)
	return wrapSingleShapeWithBounds(klein, objDef)
}

func parseTriangle(objDef map[string]interface{}) ([]shape.Shape, error) {
	p1, err := utils.RequiredVec(objDef, "p1", utils.Dimension)
	if err != nil {
		return nil, err
	}

	p2, err := utils.RequiredVec(objDef, "p2", utils.Dimension)
	if err != nil {
		return nil, err
	}

	p3, err := utils.RequiredVec(objDef, "p3", utils.Dimension)
	if err != nil {
		return nil, err
	}

	triangle := shape.NewTriangle(p1, p2, p3)
	return wrapSingleShapeWithBounds(triangle, objDef)
}

func parseQuadraticEquation(objDef map[string]interface{}) ([]shape.Shape, error) {
	a, err := utils.RequiredFloat64SliceField(objDef, "a", 9)
	if err != nil {
		return nil, err
	}

	b, err := utils.RequiredFloat64SliceField(objDef, "b", utils.Dimension)
	if err != nil {
		return nil, err
	}

	c, err := utils.RequiredFloat64Field(objDef, "c")
	if err != nil {
		return nil, err
	}

	equation := shape.NewQuadraticEquation(mat.NewDense(3, 3, a), utils.NewVec(b), c)
	return wrapSingleShapeWithBounds(equation, objDef)
}

func parseCubicEquation(objDef map[string]interface{}) ([]shape.Shape, error) {
	a, err := requiredPolynomialCoefficients(objDef, 3)
	if err != nil {
		return nil, err
	}

	equation := shape.NewCubicEquation(a)
	return wrapSingleShapeWithBounds(equation, objDef)
}

func parseFourOrderEquation(objDef map[string]interface{}) ([]shape.Shape, error) {
	center, scale, err := parsePolynomialCenterScale(objDef)
	if err != nil {
		return nil, err
	}
	basis, err := parsePolynomialSurfaceBasis(objDef, utils.Dimension)
	if err != nil {
		return nil, err
	}

	a, err := requiredPolynomialCoefficients(objDef, 4)
	if err != nil {
		return nil, err
	}

	equation := shape.NewFourOrderEquation(a)
	equation.Center = center[:]
	equation.Scale = scale[:]
	equation.Basis = basis
	return wrapSingleShapeWithBounds(equation, objDef)
}

func finishEngravableShape(s shape.Shape, objDef map[string]interface{}) ([]shape.Shape, error) {
	if err := applyEngravingFunc(s, objDef); err != nil {
		return nil, err
	}

	return wrapSingleShapeWithBounds(s, objDef)
}

func wrapSingleShapeWithBounds(s shape.Shape, objDef map[string]interface{}) ([]shape.Shape, error) {
	return wrapShapesWithBounds([]shape.Shape{s}, objDef)
}

func wrapShapesWithBounds(shapes []shape.Shape, objDef map[string]interface{}) ([]shape.Shape, error) {
	bounds, ok, err := parseShapeBounds(objDef)
	if err != nil || !ok {
		return shapes, err
	}

	wrapped := make([]shape.Shape, len(shapes))
	for i, inner := range shapes {
		wrapped[i] = shape.NewBoundedShape(inner, bounds)
	}
	return wrapped, nil
}

func parseShapeBounds(objDef map[string]interface{}) (*shape.Cuboid, bool, error) {
	boundsDef, ok, err := utils.OptionalMapField(objDef, "bounds")
	if err != nil || !ok {
		return nil, ok, err
	}

	if center, hasPosition, err := utils.OptionalFloat64SliceField(boundsDef, "center", utils.Dimension); err != nil {
		return nil, true, err
	} else if hasPosition {
		size, err := utils.RequiredFloat64SliceField(boundsDef, "size", utils.Dimension)
		if err != nil {
			return nil, true, err
		}
		if err := validatePositiveBoundsSize(size); err != nil {
			return nil, true, err
		}

		positionVec := mat.NewVecDense(len(center), center)
		halfSize := maths.ScaleVec2(0.5, mat.NewVecDense(len(size), size))
		pmax := mat.NewVecDense(positionVec.Len(), nil)
		pmin := mat.NewVecDense(positionVec.Len(), nil)
		return shape.NewCuboid(
			maths.SubVec(pmin, positionVec, halfSize),
			maths.AddVec(pmax, positionVec, halfSize),
		), true, nil
	}

	pmin, err := utils.RequiredFloat64SliceField(boundsDef, "pmin", utils.Dimension)
	if err != nil {
		return nil, true, fmt.Errorf("bounds requires either position+size or pmin+pmax: %w", err)
	}
	pmax, err := utils.RequiredFloat64SliceField(boundsDef, "pmax", utils.Dimension)
	if err != nil {
		return nil, true, fmt.Errorf("bounds requires either position+size or pmin+pmax: %w", err)
	}
	if err := validateBoundsMinMax(pmin, pmax); err != nil {
		return nil, true, err
	}
	return shape.NewCuboid(
		mat.NewVecDense(len(pmin), pmin),
		mat.NewVecDense(len(pmax), pmax),
	), true, nil
}

func validatePositiveBoundsSize(size []float64) error {
	for i, value := range size {
		if value <= 0 {
			return fmt.Errorf("bounds size index %d must be > 0", i)
		}
	}
	return nil
}

func validateBoundsMinMax(pmin, pmax []float64) error {
	for i := range pmin {
		if pmin[i] >= pmax[i] {
			return fmt.Errorf("bounds pmin index %d must be < pmax", i)
		}
	}
	return nil
}

func parsePolynomialCenterScale(objDef map[string]interface{}) ([3]float64, [3]float64, error) {
	center, hasCenter, err := utils.OptionalFloat64SliceField(objDef, "center", utils.Dimension)
	if err != nil {
		return [3]float64{}, [3]float64{}, err
	}

	scale, err := parsePolynomialScale(objDef)
	if err != nil {
		return [3]float64{}, [3]float64{}, err
	}
	if err := validatePolynomialScale(scale); err != nil {
		return [3]float64{}, [3]float64{}, err
	}
	if !hasCenter {
		center = nil
	}
	return normalizePolynomialCenterScale(center, scale)
}

func parsePolynomialScale(objDef map[string]interface{}) ([]float64, error) {
	value, ok := objDef["scale"]
	if !ok {
		return nil, nil
	}

	if values, err := utils.ToFloat64Slice(value); err == nil {
		if err := utils.RequireSliceLength("scale", values, utils.Dimension); err != nil {
			return nil, err
		}
		return values, nil
	}

	scale, err := utils.RequiredFloat64Field(map[string]interface{}{"scale": value}, "scale")
	if err != nil {
		return nil, err
	}
	return []float64{scale, scale, scale}, nil
}

func validatePolynomialScale(scale []float64) error {
	for i, value := range scale {
		if value <= 0 {
			return fmt.Errorf("scale index %d must be > 0", i)
		}
	}
	return nil
}

func ParseShapeForSTL(objDef map[string]interface{}) ([]shape.Shape, error) {
	filePath, err := utils.RequiredStringField(objDef, "file")
	if err != nil {
		return nil, err
	}
	center, err := utils.RequiredFloat64SliceField(objDef, "center", utils.Dimension)
	if err != nil {
		return nil, err
	}
	zDir, err := utils.RequiredFloat64SliceField(objDef, "z_dir", utils.Dimension)
	if err != nil {
		return nil, err
	}
	xDir, err := utils.RequiredFloat64SliceField(objDef, "x_dir", utils.Dimension)
	if err != nil {
		return nil, err
	}
	scale, err := utils.RequiredFloat64SliceField(objDef, "scale", utils.Dimension)
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

func applyEngravingFunc(target interface{ shape.Shape }, objDef map[string]interface{}) error {
	value, ok, err := utils.OptionalStringField(objDef, "engraving_func")
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	engravingFunc, exists := example_lib.EngravingFuncMap[value]
	if !exists {
		return fmt.Errorf("unknown engraving_func %q", value)
	}
	switch shaped := target.(type) {
	case *shape.Cuboid:
		shaped.EngravingFunc = engravingFunc
	case *shape.Sphere:
		shaped.EngravingFunc = engravingFunc
	}
	return nil
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
