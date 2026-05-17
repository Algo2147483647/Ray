package controller

import (
	modelcamera "github.com/Algo2147483647/ray/engine/go/internal/model/camera"
	"testing"
)

func TestParseCamerasSupportsConfiguredCamera(t *testing.T) {
	script := &Script{
		Cameras: []CameraScript{
			{
				Position:    []float64{1, 2, 3},
				LookAt:      []float64{4, 5, 6},
				Up:          []float64{0, 0, 1},
				Width:       640,
				Height:      360,
				FieldOfView: 75,
			},
		},
	}

	cameras, err := ParseCameras(script)
	if err != nil {
		t.Fatalf("ParseCameras returned error: %v", err)
	}
	if len(cameras) != 1 {
		t.Fatalf("expected 1 camera, got %d", len(cameras))
	}

	camera3D, ok := cameras[0].(*modelcamera.Camera3D)
	if !ok {
		t.Fatalf("expected *camera.Camera3D, got %T", cameras[0])
	}
	if camera3D.Width != 640 || camera3D.Height != 360 {
		t.Fatalf("unexpected camera size: %dx%d", camera3D.Width, camera3D.Height)
	}
	if camera3D.FieldOfView != 75 {
		t.Fatalf("unexpected field of view: %v", camera3D.FieldOfView)
	}
	if camera3D.Direction == nil {
		t.Fatal("expected camera direction to be set from look_at")
	}
}
