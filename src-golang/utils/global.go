package utils

import "src-golang/model/optics"

const (
	IsDebug = false
)

func DebugIsRecordRay(ray *optics.Ray, row, col, sample int) {
	if row%100 == 1 && col%100 == 1 && sample == 0 {
		ray.DebugSwitch = true
	} else {
		ray.DebugSwitch = false
	}
}
