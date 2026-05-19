package factory

import "github.com/Algo2147483647/ray/engine/sceneio/parser"

var (
	requiredStringField       = parser.RequiredStringField
	optionalStringField       = parser.OptionalStringField
	optionalBoolField         = parser.OptionalBoolField
	optionalFloat64Field      = parser.OptionalFloat64Field
	requiredFloat64Field      = parser.RequiredFloat64Field
	requiredFloat64SliceField = parser.RequiredFloat64SliceField
	optionalFloat64SliceField = parser.OptionalFloat64SliceField
	requireSliceLength        = parser.RequireSliceLength
	toFloat64Slice            = parser.ToFloat64Slice
)
