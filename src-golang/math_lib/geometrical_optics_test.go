package math_lib

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
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
