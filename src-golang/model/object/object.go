package object

import "src-golang/model/object/shape"

// Object 表示场景中的物体
type Object struct {
	Shape    shape.Shape // 几何形状
	Material *Material   // 材质属性
}
