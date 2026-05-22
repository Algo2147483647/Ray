package controller

import (
	"testing"

	"github.com/Algo2147483647/ray/engine/controller/parser"
)

func TestResolveRenderConfigAcceptsWorkingSpaceAlias(t *testing.T) {
	config := ResolveRenderConfig(&parser.Script{
		Render: parser.RenderScript{
			WorkingSpace: "acescg",
		},
	}, RenderOverrides{})

	if config.ColorSpace != "acescg" {
		t.Fatalf("expected working_space alias to set color space, got %q", config.ColorSpace)
	}
}

func TestResolveRenderConfigPrefersColorSpaceOverAlias(t *testing.T) {
	config := ResolveRenderConfig(&parser.Script{
		Render: parser.RenderScript{
			ColorSpace:   "xyz",
			WorkingSpace: "acescg",
		},
	}, RenderOverrides{})

	if config.ColorSpace != "xyz" {
		t.Fatalf("expected color_space to win over working_space alias, got %q", config.ColorSpace)
	}
}
