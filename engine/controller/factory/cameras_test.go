package factory

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
	"github.com/Algo2147483647/ray/engine/utils"
)

func TestBuildCamera3DAcceptsCanonicalDirection(t *testing.T) {
	utils.SetDimension(3)

	cam, err := BuildCamera3DFromScript(parser.CameraScript{
		Position:    []float64{0, -3, 1},
		Direction:   []float64{0, 3, -1},
		Up:          []float64{0, 0, 1},
		FieldOfView: 60,
		AspectRatio: 1,
	})
	if err != nil {
		t.Fatalf("build canonical camera: %v", err)
	}
	if cam.Direction == nil || cam.Direction.Len() != 3 {
		t.Fatalf("expected normalized 3D direction, got %#v", cam.Direction)
	}
}

func TestBuildCameraFromScriptRejectsNonStandardCameraType(t *testing.T) {
	utils.SetDimension(3)

	_, err := BuildCameraFromScript(parser.CameraScript{
		Type:        modelcamera.CameraType("camera3d"),
		Position:    []float64{0, -3, 1},
		Direction:   []float64{0, 3, -1},
		Up:          []float64{0, 0, 1},
		FieldOfView: 60,
		AspectRatio: 1,
	})
	if err == nil {
		t.Fatalf("expected non-standard camera type to be rejected")
	}
}
