package model

import "github.com/Algo2147483647/ray/engine/model/camera"

// RenderTarget binds an imaging model to one independent sampling/storage
// surface and its destination. Cameras remain reusable scene descriptions.
type RenderTarget struct {
	Camera camera.RayCamera
	Film   *camera.Film
	Output string
}
