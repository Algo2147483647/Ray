package controller

import (
	"bufio"
	"fmt"
	"github.com/spf13/cast"
	"gonum.org/v1/gonum/mat"
	"os"
	"src-golang/math_lib"
	"src-golang/model/shape"
	"src-golang/utils/example_lib"
	"strings"
)

func ParseShape(objDef map[string]interface{}) shape.Shape {
	switch objDef["shape"] {
	case "cuboid":
		res := &shape.Cuboid{}
		if _, ok := objDef["position"]; ok {
			position := mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"]))
			halfSize := math_lib.ScaleVec2(0.5, mat.NewVecDense(3, cast.ToFloat64Slice(objDef["size"])))
			pmax := mat.NewVecDense(3, nil)
			pmin := mat.NewVecDense(3, nil)
			pmax.AddVec(position, halfSize)
			pmin.SubVec(position, halfSize)
			res = shape.NewCuboid(pmin, pmax)
		}

		if _, ok := objDef["pmax"]; ok {
			res = shape.NewCuboid(
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmin"])),
				mat.NewVecDense(3, cast.ToFloat64Slice(objDef["pmax"])),
			)
		}

		if _, ok := objDef["engraving_func"]; ok {
			res.EngravingFunc = example_lib.EngravingFuncMap[cast.ToString(objDef["engraving_func"])]
		}
		return res

	case "sphere":
		res := shape.NewSphere(
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["position"])),
			cast.ToFloat64(objDef["r"]),
		)
		if _, ok := objDef["engraving_func"]; ok {
			res.EngravingFunc = example_lib.EngravingFuncMap[cast.ToString(objDef["engraving_func"])]
		}
		return res

	case "triangle":
		return shape.NewTriangle(
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p1"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p2"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["p3"])),
		)

	case "plane":
	case "quadratic equation":
		return shape.NewQuadraticEquation(
			mat.NewDense(3, 3, cast.ToFloat64Slice(objDef["a"])),
			mat.NewVecDense(3, cast.ToFloat64Slice(objDef["b"])),
			cast.ToFloat64(objDef["c"]),
		)

	case "stl":
		return ParseShapeForSTL(objDef)
	}

	return nil
}

func ParseShapeForSTL(objDef map[string]interface{}) {
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

	var triangles []shape.Shape

	// 判断STL文件类型（ASCII还是二进制）
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	firstLine := scanner.Text()

	// 重置文件指针
	file.Seek(0, 0)

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
					triangles = append(triangles, shape.NewTriangle(
						transformVertex(p1, rotationMatrix, translation),
						transformVertex(p2, rotationMatrix, translation),
						transformVertex(p3, rotationMatrix, translation),
					))
					p1, p2, p3 = nil, nil, nil
				}
			}
		}
	} else {
		// 二进制STL文件处理（简化版）
		// 跳过文件头（80字节）
		reader := bufio.NewReader(file)
		reader.Discard(80)

		// 读取三角形数量（4字节）
		// 注意：这里需要实际实现二进制读取逻辑
		// 为简化起见，我们只处理ASCII STL
		panic("Binary STL not implemented yet")
	}
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
