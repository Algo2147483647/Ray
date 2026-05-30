package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
)

func TestResolveRenderConfigAcceptsFilmColorSpaceAlias(t *testing.T) {
	config := ResolveRenderConfig(&parser.Script{
		Render: parser.RenderScript{
			FilmColorSpace: "acescg",
		},
	}, RenderOverrides{CameraIndex: -1})

	if config.ColorSpace != "acescg" {
		t.Fatalf("expected working_space alias to set color space, got %q", config.ColorSpace)
	}
}

func TestResolveRenderConfigPrefersColorSpaceOverAlias(t *testing.T) {
	config := ResolveRenderConfig(&parser.Script{
		Render: parser.RenderScript{
			ColorSpace:     "xyz",
			FilmColorSpace: "acescg",
		},
	}, RenderOverrides{CameraIndex: -1})

	if config.ColorSpace != "xyz" {
		t.Fatalf("expected color_space to win over working_space alias, got %q", config.ColorSpace)
	}
}

func TestParseRenderOverridesAcceptsRepeatedScripts(t *testing.T) {
	overrides, err := ParseRenderOverrides([]string{
		"--script", "studio.json",
		"--script", "geometry.json",
	})
	if err != nil {
		t.Fatalf("parse overrides: %v", err)
	}

	if len(overrides.ScriptPaths) != 2 {
		t.Fatalf("expected two script paths, got %v", overrides.ScriptPaths)
	}
	if overrides.ScriptPath != "studio.json" {
		t.Fatalf("expected first script path to remain primary, got %q", overrides.ScriptPath)
	}
}

func TestResolveRenderConfigsExpandsRenderJobs(t *testing.T) {
	configs := ResolveRenderConfigs(&parser.Script{
		Render: parser.RenderScript{
			Samples:     8,
			Width:       320,
			OutputImage: "base.png",
		},
		Renders: []parser.RenderScript{
			{OutputImage: "front.png"},
			{Samples: 32, OutputImage: "detail.png"},
		},
	}, RenderOverrides{CameraIndex: -1})

	if len(configs) != 2 {
		t.Fatalf("expected two render configs, got %d", len(configs))
	}
	if configs[0].Samples != 8 || configs[0].Width != 320 || configs[0].OutputImage != "front.png" {
		t.Fatalf("unexpected first render config: %+v", configs[0])
	}
	if configs[1].Samples != 32 || configs[1].Width != 320 || configs[1].OutputImage != "detail.png" {
		t.Fatalf("unexpected second render config: %+v", configs[1])
	}
}

func TestResolveRenderConfigsRenderJobInheritsCameraIndexWhenOmitted(t *testing.T) {
	configs := ResolveRenderConfigs(&parser.Script{
		Render: parser.RenderScript{
			CameraIndex:    2,
			CameraIndexSet: true,
		},
		Renders: []parser.RenderScript{
			{OutputImage: "inherited.png"},
			{CameraIndex: 0, CameraIndexSet: true, OutputImage: "override.png"},
		},
	}, RenderOverrides{CameraIndex: -1})

	if configs[0].CameraIndex != 2 {
		t.Fatalf("expected first render job to inherit camera index 2, got %d", configs[0].CameraIndex)
	}
	if configs[1].CameraIndex != 0 {
		t.Fatalf("expected second render job to override camera index to 0, got %d", configs[1].CameraIndex)
	}
}
