package optics

import (
	"gonum.org/v1/gonum/mat"
	"math"
	"src-golang/math_lib"
	"testing"
)

// TestConvertToMonochromeMonteCarlo 使用蒙特卡洛方法测试 ConvertToMonochrome 方法
func TestConvertToMonochromeMonteCarlo(t *testing.T) {
	const numSamples = 500000

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
