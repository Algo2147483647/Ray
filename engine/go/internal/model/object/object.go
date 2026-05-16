package object

import (
	"github.com/Algo2147483647/ray/engine/go/internal/model/optics"
	"github.com/Algo2147483647/ray/engine/go/internal/model/shape"
)

// Object 表示场景中的物体
type Object struct {
	Shape    shape.Shape      // 几何形状
	Material *optics.Material // 材质属性
}
