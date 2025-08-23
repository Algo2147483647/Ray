package math_lib

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"image/color"
	"math"
	"testing"
)

// TestRefractVisualization 测试折射函数并可视化不同入射角度的折射效果 (单测有问题)
func TestRefractVisualization(t *testing.T) {
	// 指定折射率，可以更改
	RefractiveIndex := 0.4

	// 创建图表
	p := plot.New()
	p.Title.Text = fmt.Sprintf("Refraction Visualization (RefractiveIndex=%.2f)", RefractiveIndex)
	p.X.Label.Text = "X"
	p.Y.Label.Text = "Y"

	// 设置坐标轴比例，确保图形不被拉伸
	p.X.Min = -1.5
	p.X.Max = 1.5
	p.Y.Min = -1.5
	p.Y.Max = 1.5

	// 绘制法线（垂直向上）
	normalLineData := plotter.XYs{{0, -1.2}, {0, 1.2}}
	normalLine, err := plotter.NewLine(normalLineData)
	if err != nil {
		t.Fatal(err)
	}
	normalLine.Color = color.RGBA{128, 128, 128, 255} // 灰色
	p.Add(normalLine)

	// 绘制界面（水平线）
	interfaceLineData := plotter.XYs{{-1.5, 0}, {1.5, 0}}
	interfaceLine, err := plotter.NewLine(interfaceLineData)
	if err != nil {
		t.Fatal(err)
	}
	interfaceLine.Color = color.RGBA{0, 0, 0, 255}                 // 黑色
	interfaceLine.Dashes = []vg.Length{vg.Points(5), vg.Points(5)} // 虚线
	p.Add(interfaceLine)

	// 生成入射光线和折射光线
	var incidentRays, refractRays plotter.XYs
	var incidentColors, refractColors []color.Color

	// 生成不同角度的入射光线
	angleStep := 15.0 // 角度步长
	for angle := -90.0; angle <= 0; angle += angleStep {
		// 计算入射光线方向（从光源指向界面）
		rad := angle * math.Pi / 180.0
		incidentRay := mat.NewVecDense(3, []float64{math.Sin(rad), -math.Cos(rad), 0})

		// 法线方向（从界面指向外）
		normal := mat.NewVecDense(3, []float64{0, -1, 0})

		// 计算折射光线
		refractedRay := Refract(incidentRay, normal, 1.0/RefractiveIndex)

		// 添加入射光线数据
		incidentRays = append(incidentRays,
			plotter.XY{X: -0.5 * math.Sin(rad), Y: 0.5 * math.Cos(rad)}, // 起点
			plotter.XY{X: 0, Y: 0}) // 终点（界面点）

		// 添加折射光线数据
		refractAngle := math.Atan2(refractedRay.AtVec(0), -refractedRay.AtVec(1))
		refractRays = append(refractRays,
			plotter.XY{X: 0, Y: 0}, // 起点（界面点）
			plotter.XY{X: 0.5 * math.Sin(refractAngle), Y: -0.5 * math.Cos(refractAngle)}) // 终点

		// 根据角度生成颜色
		normalizedAngle := (angle + 90.0) / 90.0 // 归一化到[0,1]
		incidentColors = append(incidentColors, getColor(normalizedAngle))
		refractColors = append(refractColors, getColor(normalizedAngle))
	}

	// 绘制入射光线
	for i := 0; i < len(incidentRays)/2; i++ {
		lineData := plotter.XYs{incidentRays[i*2], incidentRays[i*2+1]}
		line, err := plotter.NewLine(lineData)
		if err != nil {
			t.Fatal(err)
		}
		line.Color = incidentColors[i]
		line.Width = vg.Points(2)
		p.Add(line)
	}

	// 绘制折射光线
	for i := 0; i < len(refractRays)/2; i++ {
		lineData := plotter.XYs{refractRays[i*2], refractRays[i*2+1]}
		line, err := plotter.NewLine(lineData)
		if err != nil {
			t.Fatal(err)
		}
		line.Color = refractColors[i]
		line.Width = vg.Points(2)
		p.Add(line)
	}

	// 保存图表为PNG文件
	if err := p.Save(10*vg.Inch, 10*vg.Inch, fmt.Sprintf("refract_visualization_RefractiveIndex_%.2f.png", RefractiveIndex)); err != nil {
		t.Fatal(err)
	}

	t.Logf("Refraction visualization saved to refract_visualization_RefractiveIndex_%.2f.png", RefractiveIndex)
}

// getColor 根据角度生成颜色
func getColor(t float64) color.Color {
	// 使用HSL到RGB的转换来生成颜色
	hue := t * 300.0 // 色相从0到300度
	saturation := 1.0
	lightness := 0.5

	// HSL转RGB
	c := (1 - math.Abs(2*lightness-1)) * saturation
	x := c * (1 - math.Abs(math.Mod(hue/60.0, 2)-1))
	m := lightness - c/2

	var r, g, b float64
	if hue < 60 {
		r, g, b = c, x, 0
	} else if hue < 120 {
		r, g, b = x, c, 0
	} else if hue < 180 {
		r, g, b = 0, c, x
	} else if hue < 240 {
		r, g, b = 0, x, c
	} else {
		r, g, b = x, 0, c
	}

	return color.RGBA{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
		A: 255,
	}
}
