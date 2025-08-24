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
			pmax.AddVec(position, halfSize)
			pmin.SubVec(position, halfSize)
			item = shape.NewCuboid(pmin, pmax)
		}

		if _, ok := objDef["pmax"]; ok {
			item = shape.NewCuboid(
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmin"])),
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmax"])),
			)
		}

		if item != nil {
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

	// 读取 STL 文件中的数据
	file, err := os.Open(file_path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// 创建变换矩阵
	// 首先根据z_dir和x_dir计算旋转矩阵
	z_dir = math_lib.Normalize(z_dir)
	x_dir = math_lib.Normalize(x_dir)
	y_dir := math_lib.Cross2(z_dir, x_dir)
	y_dir = math_lib.Normalize(y_dir)

	// 构建旋转矩阵
	rotationMatrix := mat.NewDense(3, 3, nil)
	for i := 0; i < 3; i++ {
		rotationMatrix.Set(i, 0, x_dir.AtVec(i))
		rotationMatrix.Set(i, 1, y_dir.AtVec(i))
		rotationMatrix.Set(i, 2, z_dir.AtVec(i))
	}

	// 构建平移向量
	translation := position

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
						transformVertex(p1, rotationMatrix, translation),
						transformVertex(p2, rotationMatrix, translation),
						transformVertex(p3, rotationMatrix, translation),
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
				transformVertex(p1, rotationMatrix, translation),
				transformVertex(p2, rotationMatrix, translation),
				transformVertex(p3, rotationMatrix, translation),
			)
			s := shape.Shape(triangle)
			triangles = append(triangles, &s)
		}
	}

	return triangles
}

// 变换顶点：应用旋转和平移
func transformVertex(vertex *mat.VecDense, rotation *mat.Dense, translation *mat.VecDense) *mat.VecDense {
	// 应用旋转: R * V
	rotated := new(mat.VecDense)
	rotated.MulVec(rotation, vertex)

	// 应用平移: R * V + T
	result := new(mat.VecDense)
	result.AddVec(rotated, translation)

	return result
}
