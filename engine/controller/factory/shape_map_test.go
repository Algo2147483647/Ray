package factory

import (
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/shape"
)

// These helpers exercise the internal numerical compilers. Production callers
// must enter through parser.ObjectSpec and ParseObjectSpecInSpace.
func ParseShape(definition map[string]interface{}) ([]shape.Shape, error) {
	return parseShapeMapInSpace(definition, geometry.DefaultSceneSpace())
}

func ParseShapeInSpace(definition map[string]interface{}, space geometry.SceneSpace) ([]shape.Shape, error) {
	return parseShapeMapInSpace(definition, space)
}
