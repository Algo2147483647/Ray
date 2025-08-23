package example_lib

import (
	"gonum.org/v1/gonum/mat"
	"math"
)

type EngravingFunc func(data map[string]interface{}) bool

var EngravingFuncMap map[string]EngravingFunc = map[string]EngravingFunc{
	"sphere1": EngravingFuncSphere1,
}

func EngravingFuncSphere1(data map[string]interface{}) bool {
	raySt := data["ray_start"].(*mat.VecDense)
	rayDir := data["ray_dir"].(*mat.VecDense)
	distance := data["distance"].(float64)
	center := data["center"].(*mat.VecDense)
	r := data["r"].(float64)

	// 计算交点位置
	intersection := mat.NewVecDense(3, nil)
	intersection.ScaleVec(distance, rayDir)
	intersection.AddVec(intersection, raySt)

	// 转换为相对于球心的坐标
	relPos := mat.NewVecDense(3, nil)
	relPos.SubVec(intersection, center)

	// 计算球面坐标（方位角和极角）, 归一化到单位球
	x, y, z := relPos.At(0, 0)/r, relPos.At(1, 0)/r, relPos.At(2, 0)/r
	azimuth := math.Atan2(y, x) // 计算方位角（0到2π）
	if azimuth < 0 {
		azimuth += 2 * math.Pi
	}
	polar := math.Acos(z) // 计算极角（0到π）

	// 螺旋条纹, 使用极角和方位角创建螺旋效果
	spiralParam := polar*5 + azimuth*3

	// 使用正弦函数创建平滑的条纹图案
	patternValue := math.Sin(spiralParam)

	// 如果patternValue大于0.3，则透射（true），否则相交（false）, 调整这个阈值可以改变图案的密度
	return patternValue > 0.3
}
