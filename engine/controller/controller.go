package controller

import (
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/model/material"
	"github.com/Algo2147483647/ray/engine/model/material/medium"
	"github.com/Algo2147483647/ray/engine/model/shape"
	"github.com/Algo2147483647/ray/engine/sceneio/factory"
	"github.com/Algo2147483647/ray/engine/sceneio/parser"
	"github.com/Algo2147483647/ray/engine/sceneio/schema"
)

type CameraScript = schema.CameraScript
type RenderScript = schema.RenderScript
type Script = schema.Script

func ReadScriptFile(filepath string) (*Script, error) {
	return parser.ReadScriptFile(filepath)
}

func LoadSceneFromScript(script *Script, scene *model.Scene) error {
	return factory.LoadSceneFromScript(script, scene)
}

func ParseCameras(script *Script) ([]camera.Camera, error) {
	return factory.ParseCameras(script)
}

func DefaultCameraScript() CameraScript {
	return factory.DefaultCameraScript()
}

func BuildCamera3DFromScript(def CameraScript) (*camera.Camera3D, error) {
	return factory.BuildCamera3DFromScript(def)
}

func BuildCameraNDimFromScript(def CameraScript) (*camera.CameraNDim, error) {
	return factory.BuildCameraNDimFromScript(def)
}

func ParseMaterials(script *Script) (map[string]*material.Material, error) {
	return factory.ParseMaterials(script)
}

func ParseMediaRegistry(script *Script) (*medium.Registry, error) {
	return factory.ParseMediaRegistry(script)
}

func GetFloat64SliceForScript(req interface{}) (int, []float64) {
	return factory.GetFloat64SliceForScript(req)
}

func ParseShape(objDef map[string]interface{}) ([]shape.Shape, error) {
	return factory.ParseShape(objDef)
}

func ParseShapeForSTL(objDef map[string]interface{}) ([]shape.Shape, error) {
	return factory.ParseShapeForSTL(objDef)
}
