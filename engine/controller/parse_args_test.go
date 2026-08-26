package controller

import (
	"strings"
	"testing"
)

func TestParseArgsRequiresScript(t *testing.T) {
	handler := NewHandler().ParseArgs(nil)
	if handler.err == nil || !strings.Contains(handler.err.Error(), "requires --script PATH") {
		t.Fatalf("expected missing --script error, got %v", handler.err)
	}
}

func TestParseArgsAcceptsSingleScriptFlag(t *testing.T) {
	handler := NewHandler().ParseArgs([]string{"--script", "scene.json"})
	if handler.err != nil {
		t.Fatalf("parse --script: %v", handler.err)
	}
	if handler.ScriptPath != "scene.json" {
		t.Fatalf("script path = %q, want scene.json", handler.ScriptPath)
	}
}

func TestParseArgsRejectsPositionalScript(t *testing.T) {
	for _, args := range [][]string{
		{"scene.json"},
		{"--script", "scene.json", "extra.json"},
	} {
		handler := NewHandler().ParseArgs(args)
		if handler.err == nil || !strings.Contains(handler.err.Error(), "does not accept positional arguments") {
			t.Fatalf("args %v: expected positional-argument error, got %v", args, handler.err)
		}
	}
}

func TestParseArgsRejectsRepeatedScriptFlag(t *testing.T) {
	handler := NewHandler().ParseArgs([]string{"--script", "first.json", "--script", "second.json"})
	if handler.err == nil || !strings.Contains(handler.err.Error(), "--script may be specified only once") {
		t.Fatalf("expected repeated --script error, got %v", handler.err)
	}
}
