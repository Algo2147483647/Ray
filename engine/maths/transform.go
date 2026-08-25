package maths

func IdentityTransform4() [4][4]float64 {
	transform := [4][4]float64{}
	for axis := range transform {
		transform[axis][axis] = 1
	}
	return transform
}
