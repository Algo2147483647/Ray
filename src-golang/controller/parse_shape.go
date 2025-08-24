package controller

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"github.com/spf13/cast"
	"gonum.org/v1/gonum/mat"
	"os"
	"src-golang/math_lib"
	"src-golang/model/shape"
	"src-golang/utils/example_lib"
	"strings"
)

func ParseShape(objDef map[string]interface{}) (res []*shape.Shape) {
	switch objDef["shape"] {
	case "cuboid":
		var item *shape.Cuboid
		if _, ok := objDef["position"]; ok {
			position := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"]))
			halfSize := math_lib.ScaleVec2(0.5, mat.NewVecDense(3, cast.ToFloat64Slice(objDef["size"])))
			pmax := mat.NewVecDense(3, nil)
			pmin := mat.NewVecDense(3, nil)
			item = shape.NewCuboid(
				math_lib.SubVec(pmin, position, halfSize),
				math_lib.AddVec(pmax, position, halfSize),
			)
		} else if _, ok := objDef["pmax"]; ok {
			item = shape.NewCuboid(
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmin"])),
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmax"])),
			)
		}

		if item == nil {
			return []*shape.Shape{}
		}

		if _, ok := objDef["engraving_func"]; ok {
			item.EngravingFunc = example_lib.EngravingFuncMap[cast.ToString(objDef["engraving_func"])]
		}
		s := shape.Shape(item)
		return []*shape.Shape{&s}

	case "sphere":
		sphere := shape.NewSphere(
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"])),
			cast.ToFloat64(objDef["r"]),
		)
		if _, ok := objDef["engraving_func"]; ok {
			sphere.EngravingFunc = example_lib.EngravingFuncMap[cast.ToString(objDef["engraving_func"])]
		}
		s := shape.Shape(sphere)
		return []*shape.Shape{&s}

	case "triangle":
		triangle := shape.NewTriangle(
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p1"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p2"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p3"])),
		)
		s := shape.Shape(triangle)
		return []*shape.Shape{&s}

	case "plane":
	case "quadratic equation":
		qe := shape.NewQuadraticEquation(
			mat.NewDense(3, 3, cast.ToFloat64Slice(objDef["a"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["b"])),
			cast.ToFloat64(objDef["c"]),
		)
		s := shape.Shape(qe)
		return []*shape.Shape{&s}

	case "stl":
		return ParseShapeForSTL(objDef)
	}

	return []*shape.Shape{}
}

func ParseShapeForSTL(objDef map[string]interface{}) []*shape.Shape {
	file_path := cast.ToString(objDef["file"])
	position := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"]))
	z_dir := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["z_dir"]))
	x_dir := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["x_dir"]))
	scale := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["scale"]))

	// 读取 STL 文件中的数据
	file, err := os.Open(file_path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// 创建变换矩阵, 构建4x4统一变换矩阵
	transformMatrix := mat.NewDense(4, 4, []float64{
		1, 0, 0, position.AtVec(0),
		0, 1, 0, position.AtVec(1),
		0, 0, 1, position.AtVec(2),
		0, 0, 0, 1,
	})

	// 首先根据z_dir和x_dir计算旋转矩阵
	z_dir = math_lib.Normalize(z_dir)
	x_dir = math_lib.Normalize(x_dir)
	y_dir := math_lib.Normalize(math_lib.Cross2(z_dir, x_dir))

	for i := 0; i < 3; i++ {
		// 构建旋转矩阵
		transformMatrix.Set(i, 0, x_dir.AtVec(i))
		transformMatrix.Set(i, 1, y_dir.AtVec(i))
		transformMatrix.Set(i, 2, z_dir.AtVec(i))
		// 应用缩放
		for j := 0; j < 3; j++ {
			transformMatrix.Set(i, j, transformMatrix.At(i, j)*scale.AtVec(j))
		}
	}

	var triangles []*shape.Shape

	// 判断STL文件类型（ASCII还是二进制）
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	firstLine := scanner.Text()
	file.Seek(0, 0) // 重置文件指针

	if strings.HasPrefix(firstLine, "solid") {
		// ASCII STL文件
		scanner := bufio.NewScanner(file)
		var p1, p2, p3 *mat.VecDense

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "vertex") {
				// 解析顶点
				var x, y, z float64
				_, err := fmt.Sscanf(line, "vertex %f %f %f", &x, &y, &z)
				if err != nil {
					panic(err)
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
					s := shape.Shape(triangle)
					triangles = append(triangles, &s)
					p1, p2, p3 = nil, nil, nil
				}
			}
		}
	} else {
		// 二进制STL文件处理
		header := make([]byte, 80) // 跳过文件头（80字节）
		_, err := file.Read(header)
		if err != nil {
			panic(err)
		}

		var numTriangles uint32
		err = binary.Read(file, binary.LittleEndian, &numTriangles) // 读取三角形数量（4字节，小端序）
		if err != nil {
			panic(err)
		}

		// 读取每个三角形的数据
		for i := uint32(0); i < numTriangles; i++ {
			normal := make([]byte, 12) // 跳过法向量（12字节，3个float32）
			_, err := file.Read(normal)
			if err != nil {
				panic(err)
			}

			var vertices [9]float32 // 读取三个顶点的坐标（36字节，9个float32）
			err = binary.Read(file, binary.LittleEndian, &vertices)
			if err != nil {
				panic(err)
			}

			var attrByteCount uint16 // 读取属性字节数（2字节），并跳过
			err = binary.Read(file, binary.LittleEndian, &attrByteCount)
			if err != nil {
				panic(err)
			}

			// 创建三角形
			p1 := mat.NewVecDense(3, []float64{float64(vertices[0]), float64(vertices[1]), float64(vertices[2])})
			p2 := mat.NewVecDense(3, []float64{float64(vertices[3]), float64(vertices[4]), float64(vertices[5])})
			p3 := mat.NewVecDense(3, []float64{float64(vertices[6]), float64(vertices[7]), float64(vertices[8])})
			triangle := shape.NewTriangle(
				transformVertexWithMatrix(p1, transformMatrix),
				transformVertexWithMatrix(p2, transformMatrix),
				transformVertexWithMatrix(p3, transformMatrix),
			)
			s := shape.Shape(triangle)
			triangles = append(triangles, &s)
		}
	}

	return triangles
}

// 使用4x4变换矩阵变换顶点
func transformVertexWithMatrix(vertex *mat.VecDense, transformMatrix *mat.Dense) *mat.VecDense {
	// 将3D顶点转换为齐次坐标（4D）
	vertexHomogeneous := mat.NewVecDense(4, []float64{
		vertex.AtVec(0),
		vertex.AtVec(1),
		vertex.AtVec(2),
		1.0,
	})

	// 应用变换矩阵
	transformed := new(mat.VecDense)
	transformed.MulVec(transformMatrix, vertexHomogeneous)

	// 转换回3D坐标
	result := mat.NewVecDense(3, []float64{
		transformed.AtVec(0),
		transformed.AtVec(1),
		transformed.AtVec(2),
	})

	return result
}
