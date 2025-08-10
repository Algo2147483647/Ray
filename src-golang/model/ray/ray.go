package ray

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
)

// Ray 表示光线
type Ray struct {
	Origin       *mat.VecDense
	Direction    *mat.VecDense
	Color        *mat.VecDense
	Refractivity float64
	DebugSwitch  bool
	DebugTraces  []map[string]interface{}
}

func (r *Ray) DebugString() string {
	res := ""
	for _, trace := range r.DebugTraces {
		lv := trace["level"].(int)
		dist := trace["distance"].(float64)
		colorVec := trace["color"].(*mat.VecDense)
		hitObj := trace["hit_object"]

		res += fmt.Sprintf("[Level %d] Hit: %-15s | Dist: %-8.2f | Color: (%.2f,%.2f,%.2f)\n",
			lv,
			hitObj,
			dist,
			colorVec.At(0, 0),
			colorVec.At(1, 0),
			colorVec.At(2, 0),
		)
	}
	return res
}
