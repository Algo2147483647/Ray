package factory

import (
	"math"
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	"github.com/Algo2147483647/ray/engine/maths/geometry"
	"github.com/Algo2147483647/ray/engine/model"
)

func TestLoadSceneFromScriptParsesGeometry(t *testing.T) {
	scene := model.NewScene()
	script := &parser.Script{
		Render:   parser.RenderScript{Dimension: 3},
		Geometry: &parser.GeometryScript{Type: "klein"},
	}

	if err := LoadSceneFromScript(script, scene); err != nil {
		t.Fatalf("LoadSceneFromScript failed: %v", err)
	}
	if scene.Geometry != geometry.Klein() {
		t.Fatalf("expected Klein geometry, got %v", scene.Geometry)
	}
	if scene.MaxArc != 0 {
		t.Fatalf("expected unbounded Klein max arc, got %f", scene.MaxArc)
	}
}

func TestLoadSceneFromScriptDefaultsSphericalMaxArc(t *testing.T) {
	scene := model.NewScene()
	script := &parser.Script{
		Render:   parser.RenderScript{Dimension: 4},
		Geometry: &parser.GeometryScript{Type: "spherical"},
	}

	if err := LoadSceneFromScript(script, scene); err != nil {
		t.Fatalf("LoadSceneFromScript failed: %v", err)
	}
	if scene.Geometry != geometry.Spherical() {
		t.Fatalf("expected spherical geometry, got %v", scene.Geometry)
	}
	if math.Abs(scene.MaxArc-2*math.Pi) > 1e-12 {
		t.Fatalf("expected default spherical max arc 2*pi, got %.15f", scene.MaxArc)
	}
}

func TestLoadSceneFromScriptResetsGeometryOnReuse(t *testing.T) {
	scene := model.NewScene()
	first := &parser.Script{
		Render:   parser.RenderScript{Dimension: 4},
		Geometry: &parser.GeometryScript{Type: "spherical"},
	}
	if err := LoadSceneFromScript(first, scene); err != nil {
		t.Fatalf("LoadSceneFromScript first failed: %v", err)
	}

	second := &parser.Script{Render: parser.RenderScript{Dimension: 3}}
	if err := LoadSceneFromScript(second, scene); err != nil {
		t.Fatalf("LoadSceneFromScript second failed: %v", err)
	}
	if scene.Geometry != nil {
		t.Fatalf("expected reused scene geometry reset to nil, got %v", scene.Geometry)
	}
	if scene.MaxArc != 0 {
		t.Fatalf("expected reused scene max arc reset to 0, got %f", scene.MaxArc)
	}
}
