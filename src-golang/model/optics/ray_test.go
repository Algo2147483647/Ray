package optics

import (
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
	"math"
	"src-golang/math_lib"
	"testing"
)

// TestConvertToMonochromeMonteCarlo 使用蒙特卡洛方法测试 ConvertToMonochrome 方法
func TestConvertToMonochromeMonteCarlo(t *testing.T) {
	const numSamples = 10000000

	// 存储收集的数据
	redValues := make([]float64, numSamples)
	greenValues := make([]float64, numSamples)
	blueValues := make([]float64, numSamples)
	wavelengths := make([]float64, numSamples)

	// 蒙特卡洛模拟
	for i := 0; i < numSamples; i++ {
		ray := &Ray{
			Color: mat.NewVecDense(3, []float64{1.0, 1.0, 1.0}),
		}
		ray.ConvertToMonochrome()

		wavelengths[i] = ray.WaveLength
		redValues[i] = ray.Color.AtVec(0)
		greenValues[i] = ray.Color.AtVec(1)
		blueValues[i] = ray.Color.AtVec(2)
	}

	// 验证分布是否合理
	validateDistribution(t, wavelengths, redValues, greenValues, blueValues)
}

// validateDistribution 验证分布是否合理
func validateDistribution(t *testing.T, wavelengths, red, green, blue []float64) {
	// 检查波长范围
	minWavelength, maxWavelength := math.Inf(1), math.Inf(-1)
	for _, w := range wavelengths {
		if w < minWavelength {
			minWavelength = w
		}
		if w > maxWavelength {
			maxWavelength = w
		}
	}

	if minWavelength < math_lib.WavelengthMin || maxWavelength > math_lib.WavelengthMax {
		t.Errorf("Wavelength out of range: [%f, %f], expected range: [%f, %f]",
			minWavelength, maxWavelength, math_lib.WavelengthMin, math_lib.WavelengthMax)
	}

	// 检查颜色值是否合理
	checkColorRange := func(values []float64, name string) {
		for _, v := range values {
			// 由于使用了 ScaleVec，颜色值可能超过 [0,1] 范围
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Errorf("Invalid %s value: %f", name, v)
			}
		}
	}

	checkColorRange(red, "red")
	checkColorRange(green, "green")
	checkColorRange(blue, "blue")

	t.Logf("Sampled %d rays", len(wavelengths))
	t.Logf("Wavelength range: [%f, %f]", minWavelength, maxWavelength)

	// 计算基本统计信息
	calculateStats := func(data []float64) (mean, std float64) {
		var sum float64
		for _, v := range data {
			sum += v
		}
		mean = sum / float64(len(data))

		var variance float64
		for _, v := range data {
			variance += (v - mean) * (v - mean)
		}
		std = math.Sqrt(variance / float64(len(data)))

		return mean, std
	}

	wMean, wStd := calculateStats(wavelengths)
	rMean, rStd := calculateStats(red)
	gMean, gStd := calculateStats(green)
	bMean, bStd := calculateStats(blue)

	t.Logf("Wavelength: mean=%f, std=%f", wMean, wStd)
	t.Logf("Red: mean=%f, std=%f", rMean, rStd)
	t.Logf("Green: mean=%f, std=%f", gMean, gStd)
	t.Logf("Blue: mean=%f, std=%f", bMean, bStd)
}

func TestWaveLengthToRGB(t *testing.T) {
	// 创建新的图表
	p := plot.New()

	p.Title.Text = "RGB values by Wavelength"
	p.X.Label.Text = "Wavelength (nm)"
	p.Y.Label.Text = "RGB Value"

	// 创建数据点
	var rData, gData, bData plotter.XYs
	step := 1.0
	count := int((math_lib.WavelengthMax - math_lib.WavelengthMin) / step)

	rData = make(plotter.XYs, count)
	gData = make(plotter.XYs, count)
	bData = make(plotter.XYs, count)

	for i := 0; i < count; i++ {
		wavelength := math_lib.WavelengthMin + float64(i)*step
		rgb := WaveLengthToRGB(wavelength)

		rData[i].X = wavelength
		rData[i].Y = rgb.AtVec(0)

		gData[i].X = wavelength
		gData[i].Y = rgb.AtVec(1)

		bData[i].X = wavelength
		bData[i].Y = rgb.AtVec(2)
	}

	// 添加数据线到图表
	rLine, err := plotter.NewLine(rData)
	if err != nil {
		t.Fatal(err)
	}
	rLine.Color = plotutil.Color(0) // Red line

	gLine, err := plotter.NewLine(gData)
	if err != nil {
		t.Fatal(err)
	}
	gLine.Color = plotutil.Color(1) // Green line

	bLine, err := plotter.NewLine(bData)
	if err != nil {
		t.Fatal(err)
	}
	bLine.Color = plotutil.Color(2) // Blue line

	// 将线条添加到图表
	p.Add(rLine, gLine, bLine)
	p.Legend.Add("Red", rLine)
	p.Legend.Add("Green", gLine)
	p.Legend.Add("Blue", bLine)

	// 设置图例位置
	p.Legend.Top = true

	// 保存图表为PNG文件
	if err := p.Save(10*vg.Inch, 6*vg.Inch, "wavelength_rgb_curve.png"); err != nil {
		t.Fatal(err)
	}
}

// Benchmark测试WaveLengthToRGB性能
func BenchmarkWaveLengthToRGB(b *testing.B) {
	// 随机生成测试波长值
	wavelengths := make([]float64, b.N)
	for i := 0; i < b.N; i++ {
		wavelengths[i] = math_lib.WavelengthMin + float64(i%int(math_lib.WavelengthMax-math_lib.WavelengthMin))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WaveLengthToRGB(wavelengths[i])
	}
}
