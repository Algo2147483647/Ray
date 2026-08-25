package factory

import (
	"math"
	"strings"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model"
	"github.com/Algo2147483647/ray/engine/model/camera"
)

func TestLoadSceneFromScriptParsesGeometry(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	script := &parser.Script{
		Dimension: 3,
		Geometry:  &parser.GeometryScript{Type: "klein"},
	}

	if err := LoadSceneFromScript(script, scene); err != nil {
		t.Fatalf("LoadSceneFromScript failed: %v", err)
	}
	if scene.Space.Geometry != geometry.Klein() {
		t.Fatalf("expected Klein geometry, got %v", scene.Space.Geometry)
	}
	if scene.Space.Dimension != 3 {
		t.Fatalf("expected scene dimension 3, got %d", scene.Space.Dimension)
	}
	if scene.MaxArc != 0 {
		t.Fatalf("expected unbounded Klein max arc, got %f", scene.MaxArc)
	}
}

func TestLoadSceneFromScriptDefaultsSphericalMaxArc(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	script := &parser.Script{
		Dimension: 4,
		Geometry:  &parser.GeometryScript{Type: "spherical"},
	}

	if err := LoadSceneFromScript(script, scene); err != nil {
		t.Fatalf("LoadSceneFromScript failed: %v", err)
	}
	if scene.Space.Geometry != geometry.Spherical() {
		t.Fatalf("expected spherical geometry, got %v", scene.Space.Geometry)
	}
	if math.Abs(scene.MaxArc-2*math.Pi) > 1e-12 {
		t.Fatalf("expected default spherical max arc 2*pi, got %.15f", scene.MaxArc)
	}
}

func TestLoadSceneFromScriptResetsGeometryOnReuse(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	first := &parser.Script{
		Dimension: 4,
		Geometry:  &parser.GeometryScript{Type: "spherical"},
	}
	if err := LoadSceneFromScript(first, scene); err != nil {
		t.Fatalf("LoadSceneFromScript first failed: %v", err)
	}

	second := &parser.Script{Dimension: 3}
	if err := LoadSceneFromScript(second, scene); err != nil {
		t.Fatalf("LoadSceneFromScript second failed: %v", err)
	}
	if scene.Space.Geometry == nil || scene.Space.Geometry.Kind() != geometry.EuclideanKind {
		t.Fatalf("expected reused scene geometry reset to explicit Euclidean, got %v", scene.Space.Geometry)
	}
	if scene.MaxArc != 0 {
		t.Fatalf("expected reused scene max arc reset to 0, got %f", scene.MaxArc)
	}
}

func TestLoadSceneFromScriptRejectsGeometryDimensionMismatch(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	script := &parser.Script{
		Dimension: 4,
		Geometry:  &parser.GeometryScript{Type: "klein"},
	}

	err := LoadSceneFromScript(script, scene)
	if err == nil || !strings.Contains(err.Error(), "requires scene dimension 3") {
		t.Fatalf("expected Klein dimension error, got %v", err)
	}
}

func TestLoadSceneFromScriptRejectsKleinCameraMismatch(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	script := &parser.Script{
		Dimension: 3,
		Geometry:  &parser.GeometryScript{Type: "klein"},
		Cameras: []parser.CameraScript{{
			ID:           "main",
			Type:         camera.CameraType3D,
			Position:     []float64{0, 0, 0},
			Coordinates:  [][]float64{{1, 0, 0}, {0, -1, 0}, {0, 0, 1}},
			FieldOfViews: []float64{60, 60},
			Film:         &camera.Film{Shape: []int{400, 400}},
		}},
	}

	err := LoadSceneFromScript(script, scene)
	if err == nil || !strings.Contains(err.Error(), "must use type \"hyperbolic\"") {
		t.Fatalf("expected Klein camera error, got %v", err)
	}
}

func TestLoadSceneFromScriptRejectsNegativeMaxArc(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	script := &parser.Script{
		Dimension: 3,
		Geometry:  &parser.GeometryScript{Type: "klein", MaxArc: -1},
	}

	err := LoadSceneFromScript(script, scene)
	if err == nil || !strings.Contains(err.Error(), "max_arc") {
		t.Fatalf("expected max_arc error, got %v", err)
	}
}

func TestLoadSceneFromScriptRejectsConflictingLegacyRenderDimensions(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	script := &parser.Script{Renders: []parser.RenderScript{
		{LegacyDimension: 3},
		{LegacyDimension: 4},
	}}

	err := LoadSceneFromScript(script, scene)
	if err == nil || !strings.Contains(err.Error(), "conflicts with scene dimension") {
		t.Fatalf("expected conflicting render dimensions to fail, got %v", err)
	}
}

func TestLoadSceneFromScriptRejectsLegacyDimensionConflictingWithScene(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	script := &parser.Script{
		Dimension: 3,
		Renders:   []parser.RenderScript{{LegacyDimension: 4}},
	}

	err := LoadSceneFromScript(script, scene)
	if err == nil || !strings.Contains(err.Error(), "conflicts with scene dimension") {
		t.Fatalf("expected legacy dimension conflict, got %v", err)
	}
}

func TestLoadSceneFromScriptBuildsDimensionedEuclideanGeometry(t *testing.T) {
	scene := model.NewScene(geometry.DefaultSceneSpace())
	if err := LoadSceneFromScript(&parser.Script{Dimension: 7}, scene); err != nil {
		t.Fatalf("load 7D Euclidean scene: %v", err)
	}
	if scene.Space.Geometry == nil || scene.Space.Geometry.Kind() != geometry.EuclideanKind {
		t.Fatalf("expected explicit Euclidean Geometry, got %v", scene.Space.Geometry)
	}
	if scene.Space.Dimension != 7 || scene.Space.Geometry.Dimension() != 7 {
		t.Fatalf("dimension mismatch: space=%d geometry=%d", scene.Space.Dimension, scene.Space.Geometry.Dimension())
	}
}
