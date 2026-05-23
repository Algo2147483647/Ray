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
	}, RenderOverrides{})

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
	}, RenderOverrides{})

	if config.ColorSpace != "xyz" {
		t.Fatalf("expected color_space to win over working_space alias, got %q", config.ColorSpace)
	}
}
