package math_lib

import (
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"math"
	"testing"
)

func TestCauchyDispersion(t *testing.T) {
	A := 1.500
	B := 50000.0
	C := 0.0
	t.Log(CauchyDispersion(WavelengthMax, A, B, C))
	t.Log(CauchyDispersion(WavelengthMin, A, B, C))
}

func TestRefractiveIndexCurve(t *testing.T) {
	// 创建图表
	p := plot.New()

	p.Title.Text = "Refractive Index vs Wavelength"
	p.X.Label.Text = "Wavelength (nm)"
	p.Y.Label.Text = "Refractive Index"

	// 玻璃材料参数 (来自 test.json 中的 glass3 材料)
	A := 1.000
	B := 200000.0
	C := 0.0

	// 生成数据点
	var data plotter.XYs
	step := 1.0
	count := int((WavelengthMax - WavelengthMin) / step)
	data = make(plotter.XYs, count)

	for i := 0; i < count; i++ {
		wavelength := WavelengthMin + float64(i)*step
		refractiveIndex := CauchyDispersion(wavelength, A, B, C)

		data[i].X = wavelength
		data[i].Y = refractiveIndex
	}

	// 创建线条
	line, err := plotter.NewLine(data)
	if err != nil {
		t.Fatal(err)
	}
	line.Color = plotutil.Color(0)

	// 将线条添加到图表
	p.Add(line)

	// 保存图表为PNG文件
	if err := p.Save(10*vg.Inch, 6*vg.Inch, "refractive_index_curve.png"); err != nil {
		t.Fatal(err)
	}

	t.Log("Refractive index curve saved to refractive_index_curve.png")
}

// 测试DiffuseReflect4D函数的数学正确性
func TestDiffuseReflect4D(t *testing.T) {
	// 设置测试用的入射光线和法向量
	incidentRay := mat.NewVecDense(4, []float64{1, 0, 0, 0})
	normal := mat.NewVecDense(4, []float64{0, 1, 0, 0})

	// 确保向量归一化
	Normalize(incidentRay)
	Normalize(normal)

	// 执行多次测试以验证统计特性
	n := 10000
	dotProducts := make([]float64, n)

	// 收集反射向量与法向量的点积（应该符合余弦分布）
	for i := 0; i < n; i++ {
		reflected := DiffuseReflect4D(incidentRay, normal)

		// 验证反射向量是单位向量
		length := mat.Norm(reflected, 2)
		if math.Abs(length-1.0) > 1e-10 {
			t.Errorf("Reflected vector is not normalized. Length: %f", length)
		}

		// 计算反射向量与法向量的点积
		dot := mat.Dot(reflected, normal)
		dotProducts[i] = dot
	}

	// 验证点积都为正（在法向量的同一侧）
	for i, dot := range dotProducts {
		if dot < 0 {
			t.Errorf("Dot product is negative at index %d: %f", i, dot)
		}
	}

	// 验证分布符合余弦分布的期望值
	// 对于余弦加权分布，期望值应该是2/3
	sum := 0.0
	for _, dot := range dotProducts {
		sum += dot
	}
	average := sum / float64(n)
	expected := 2.0 / 3.0
	if math.Abs(average-expected) > 0.05 { // 允许一定误差
		t.Errorf("Average dot product %f is not close to expected value %f", average, expected)
	}

	// 验证所有点积值都在[0,1]范围内
	for i, dot := range dotProducts {
		if dot < 0 || dot > 1 {
			t.Errorf("Dot product at index %d is out of range [0,1]: %f", i, dot)
		}
	}

	t.Logf("Average dot product: %f (expected: %f)", average, expected)
}

// 测试DiffuseReflect4D在不同入射角度下的行为
func TestDiffuseReflect4DWithDifferentAngles(t *testing.T) {
	// 测试不同入射角度
	testCases := []struct {
		name     string
		incident []float64
		normal   []float64
	}{
		{
			name:     "正面入射",
			incident: []float64{-1, 0, 0, 0}, // 沿x轴负方向
			normal:   []float64{1, 0, 0, 0},  // 沿x轴正方向
		},
		{
			name:     "斜入射",
			incident: []float64{-1, -1, 0, 0}, // 沿xy平面上的对角线
			normal:   []float64{0, 1, 0, 0},   // 沿y轴正方向
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			incidentRay := mat.NewVecDense(4, tc.incident)
			normal := mat.NewVecDense(4, tc.normal)

			Normalize(incidentRay)
			Normalize(normal)

			// 执行多次测试
			n := 1000
			for i := 0; i < n; i++ {
				reflected := DiffuseReflect4D(incidentRay, normal)

				// 验证反射向量是单位向量
				length := mat.Norm(reflected, 2)
				if math.Abs(length-1.0) > 1e-10 {
					t.Errorf("Reflected vector is not normalized. Length: %f", length)
				}

				// 验证反射向量与法向量的点积为正
				dot := mat.Dot(reflected, normal)
				if dot < 0 {
					t.Errorf("Reflected vector points away from surface. Dot product: %f", dot)
				}
			}
		})
	}
}
