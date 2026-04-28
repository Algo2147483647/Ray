package controller

import (
	"bufio"
	"encoding/binary"
	"fmt"
	math_lib "github.com/Algo2147483647/golang_toolkit/math/linear_algebra"
	"gonum.org/v1/gonum/mat"
	"os"
	"src-golang/model/shape"
	"src-golang/utils"
	"src-golang/utils/example_lib"
	"strings"
)

func GetFloat64SliceForScript(req interface{}) (int, []float64) {
	values, _ := toFloat64Slice(req)
	return len(values), values
}

func ParseShape(objDef map[string]interface{}) ([]shape.Shape, error) {
	shapeName, err := requiredStringField(objDef, "shape")
	if err != nil {
		return nil, err
	}

	switch shapeName {
	case "cuboid":
		if position, hasPosition, err := optionalFloat64SliceField(objDef, "position", utils.Dimension); err != nil {
			return nil, err
		} else if hasPosition {
			size, err := requiredFloat64SliceField(objDef, "size", utils.Dimension)
			if err != nil {
				return nil, err
			}

			positionVec := mat.NewVecDense(len(position), position)
			halfSize := math_lib.ScaleVec2(0.5, mat.NewVecDense(len(size), size))
			pmax := mat.NewVecDense(positionVec.Len(), nil)
			pmin := mat.NewVecDense(positionVec.Len(), nil)
			cuboid := shape.NewCuboid(
				math_lib.SubVec(pmin, positionVec, halfSize),
				math_lib.AddVec(pmax, positionVec, halfSize),
			)
			if err := applyEngravingFunc(cuboid, objDef); err != nil {
				return nil, err
			}
			return []shape.Shape{cuboid}, nil
		}

		pmin, err := requiredFloat64SliceField(objDef, "pmin", utils.Dimension)
		if err != nil {
			return nil, fmt.Errorf("cuboid requires either position+size or pmin+pmax: %w", err)
		}
		pmax, err := requiredFloat64SliceField(objDef, "pmax", utils.Dimension)
		if err != nil {
			return nil, fmt.Errorf("cuboid requires either position+size or pmin+pmax: %w", err)
		}

		cuboid := shape.NewCuboid(
			mat.NewVecDense(len(pmin), pmin),
			mat.NewVecDense(len(pmax), pmax),
		)
		if err := applyEngravingFunc(cuboid, objDef); err != nil {
			return nil, err
		}
		return []shape.Shape{cuboid}, nil

	case "sphere":
		position, err := requiredFloat64SliceField(objDef, "position", utils.Dimension)
		if err != nil {
			return nil, err
		}
		radius, err := requiredFloat64Field(objDef, "r")
		if err != nil {
			return nil, err
		}
		if radius <= 0 {
			return nil, fmt.Errorf("field %q must be > 0", "r")
		}

		sphere := shape.NewSphere(mat.NewVecDense(len(position), position), radius)
		if err := applyEngravingFunc(sphere, objDef); err != nil {
			return nil, err
		}
		return []shape.Shape{sphere}, nil

	case "triangle":
		p1, err := requiredFloat64SliceField(objDef, "p1", utils.Dimension)
		if err != nil {
			return nil, err
		}
		p2, err := requiredFloat64SliceField(objDef, "p2", utils.Dimension)
		if err != nil {
			return nil, err
		}
		p3, err := requiredFloat64SliceField(objDef, "p3", utils.Dimension)
		if err != nil {
			return nil, err
		}

		triangle := shape.NewTriangle(
			mat.NewVecDense(len(p1), p1),
			mat.NewVecDense(len(p2), p2),
			mat.NewVecDense(len(p3), p3),
		)
		return []shape.Shape{triangle}, nil

	case "plane":
		return nil, fmt.Errorf("shape %q is declared but not implemented", shapeName)

	case "quadratic equation":
		a, err := requiredFloat64SliceField(objDef, "a", 9)
		if err != nil {
			return nil, err
		}
		b, err := requiredFloat64SliceField(objDef, "b", utils.Dimension)
		if err != nil {
			return nil, err
		}
		c, err := requiredFloat64Field(objDef, "c")
		if err != nil {
			return nil, err
		}

		equation := shape.NewQuadraticEquation(
			mat.NewDense(3, 3, a),
			mat.NewVecDense(len(b), b),
			c,
		)
		return []shape.Shape{equation}, nil

	case "four-order equation":
		a, err := requiredFloat64SliceField(objDef, "a", 256)
		if err != nil {
			return nil, err
		}
		equation := shape.NewFourOrderEquation(a)
		return []shape.Shape{equation}, nil

	case "stl":
		return ParseShapeForSTL(objDef)

	default:
		return nil, fmt.Errorf("unsupported shape %q", shapeName)
	}
}

func ParseShapeForSTL(objDef map[string]interface{}) ([]shape.Shape, error) {
	filePath, err := requiredStringField(objDef, "file")
	if err != nil {
		return nil, err
	}
	position, err := requiredFloat64SliceField(objDef, "position", utils.Dimension)
	if err != nil {
		return nil, err
	}
	zDir, err := requiredFloat64SliceField(objDef, "z_dir", utils.Dimension)
	if err != nil {
		return nil, err
	}
	xDir, err := requiredFloat64SliceField(objDef, "x_dir", utils.Dimension)
	if err != nil {
		return nil, err
	}
	scale, err := requiredFloat64SliceField(objDef, "scale", utils.Dimension)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open STL file %q: %w", filePath, err)
	}
	defer file.Close()

	positionVec := mat.NewVecDense(len(position), position)
	zDirVec := math_lib.Normalize(mat.NewVecDense(len(zDir), zDir))
	xDirVec := math_lib.Normalize(mat.NewVecDense(len(xDir), xDir))
	scaleVec := mat.NewVecDense(len(scale), scale)

	transformMatrix := mat.NewDense(4, 4, []float64{
		1, 0, 0, positionVec.AtVec(0),
		0, 1, 0, positionVec.AtVec(1),
		0, 0, 1, positionVec.AtVec(2),
		0, 0, 0, 1,
	})

	yDir := math_lib.Normalize(math_lib.Cross2(zDirVec, xDirVec))

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
	value, ok, err := optionalStringField(objDef, "engraving_func")
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
