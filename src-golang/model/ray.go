package model

import "gonum.org/v1/gonum/spatial/r3"

// Ray 表示光线
type Ray struct {
	Origin       r3.Vec
	Direction    r3.Vec
	Color        r3.Vec
	Refractivity float64
}

// 光线操作函数 (在几何光学包中实现)
func DiffuseReflect(dir, norm r3.Vec) r3.Vec {
	// 实现略
	return r3.Vec{}
}

func Reflect(dir, norm r3.Vec) r3.Vec {
	// 实现略
	return r3.Vec{}
}

func Refract(dir, norm r3.Vec, index float64) r3.Vec {
	// 实现略
	return r3.Vec{}
}
