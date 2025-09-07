package controller

import (
	"github.com/spf13/cast"
	"gonum.org/v1/gonum/mat"
	"reflect"
	"src-golang/model/optics"
)

func ParseMaterials(script *Script) map[string]*optics.Material {
	materials := make(map[string]*optics.Material)

	for _, matDef := range script.Materials {
		material := optics.NewMaterial(mat.NewVecDense(GetFloat64SliceForScript(matDef["color"])))

		if val, ok := matDef["radiate"]; ok {
			material.Radiation = cast.ToBool(val)
		}
		if val, ok := matDef["radiation_type"]; ok {
			material.RadiationType = cast.ToString(val)
		}
		if val, ok := matDef["reflectivity"]; ok {
			material.Reflectivity = cast.ToFloat64(val)
		}
		if val, ok := matDef["refractivity"]; ok {
			material.Refractivity = cast.ToFloat64(val)
		}
		if val, ok := matDef["refractive_index"]; ok {
			if reflect.TypeOf(val).Kind() == reflect.Slice {
				material.RefractiveIndex = mat.NewVecDense(GetFloat64SliceForScript(val))
			} else {
				material.RefractiveIndex = mat.NewVecDense(1, []float64{cast.ToFloat64(val)})
			}
		}
		if val, ok := matDef["diffuse_loss"]; ok {
			material.DiffuseLoss = cast.ToFloat64(val)
		}
		if val, ok := matDef["reflect_loss"]; ok {
			material.ReflectLoss = cast.ToFloat64(val)
		}
		if val, ok := matDef["refract_loss"]; ok {
			material.RefractLoss = cast.ToFloat64(val)
		}
		if val, ok := matDef["color_func"]; ok {
			material.ColorFunc = optics.ColorFuncMap[cast.ToString(val)]
		}
		materials[cast.ToString(matDef["id"])] = material
	}

	return materials
}
