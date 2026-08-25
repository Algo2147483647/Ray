package factory

import (
	"encoding/json"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model/shape"
)

// These helpers exercise the internal numerical compilers. Production callers
// must enter through parser.ObjectSpec and ParseObjectSpecInSpace.
func ParseShape(definition map[string]interface{}) ([]shape.Shape, error) {
	return ParseShapeInSpace(definition, geometry.DefaultSceneSpace())
}

func ParseShapeInSpace(definition map[string]interface{}, space geometry.SceneSpace) ([]shape.Shape, error) {
	if _, ok := definition["material_id"]; !ok {
		definition = cloneTestShapeDefinition(definition)
		definition["material_id"] = "test"
	}
	data, err := json.Marshal(definition)
	if err != nil {
		return nil, err
	}
	var spec parser.ObjectSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, err
	}
	return ParseObjectSpecInSpace(spec, space)
}

func cloneTestShapeDefinition(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(source)+1)
	for key, value := range source {
		result[key] = value
	}
	return result
}
