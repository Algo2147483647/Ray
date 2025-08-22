package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"src-golang/math_lib"
	"src-golang/model/object"
	"src-golang/model/optics"
	"src-golang/utils"
)

// TracePixel 追踪单个像素
func (h *Handler) TracePixel(camera *optics.Camera, objTree *object.ObjectTree, row, col, samples int) *mat.VecDense {
	color := mat.NewVecDense(3, nil)
	for s := 0; s < samples; s++ {
		// new ray
		ray := h.RayPool.Get().(*optics.Ray)
		defer h.RayPool.Put(ray)

		// build ray
		camera.GenerateRay(ray, row, col)
		if utils.IsDebug {
			utils.DebugIsRecordRay(ray, row, col, s)
		}

		// trace ray
		sampleColor := h.TraceRay(objTree, ray, 0)
		color.AddVec(color, sampleColor)
	}
	return math_lib.ScaleVec(color, 1.0/float64(samples), color)
}
