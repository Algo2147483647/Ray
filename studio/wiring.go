package main

import (
	"github.com/Algo2147483647/ray/studio/adapt"
	"github.com/Algo2147483647/ray/studio/schema"
	"github.com/Algo2147483647/ray/studio/storage"
)

type studioScript = schema.StudioScript
type studioRenderScript = schema.StudioRenderScript
type studioCameraScript = schema.StudioCameraScript
type engineCameraScript = schema.EngineCameraScript
type intermediateScript = schema.IntermediateScript

func adaptScript(script *studioScript, source []string, dimension int) (*intermediateScript, error) {
	return adapt.AdaptScript(script, source, dimension)
}

func readStudioScriptFiles(paths []string) (*studioScript, error) {
	return storage.ReadStudioScriptFiles(paths)
}

func writeIntermediateScript(script *intermediateScript, source []string) (string, error) {
	return storage.WriteIntermediateScript(script, source)
}

func repoRoot() (string, error) {
	return storage.RepoRoot()
}

func stringField(object map[string]interface{}, key string) (string, bool) {
	return adapt.StringField(object, key)
}
