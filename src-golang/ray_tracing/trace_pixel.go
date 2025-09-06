package ray_tracing

import (
	"gonum.org/v1/gonum/mat"
	"src-golang/math_lib"
	"src-golang/model/camera"
	"src-golang/model/object"
	"src-golang/model/optics"
)

// TracePixel 追踪单个像素
func (h *Handler) TracePixel(camera camera.Camera, objTree *object.ObjectTree, samples int64, index ...int) *mat.VecDense {
	color := mat.NewVecDense(3, nil)
	ray := h.RayPool.Get().(*optics.Ray) // new ray
	defer h.RayPool.Put(ray)

	for s := int64(0); s < samples; s++ {
		camera.GenerateRay(ray, index...)          // build ray
		sampleColor := h.TraceRay(objTree, ray, 0) // trace ray
		color.AddVec(color, sampleColor)
	}
	return math_lib.ScaleVec(color, 1.0/float64(samples), color)
}
