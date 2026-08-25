package controller

import (
	"strings"
	"testing"
)

func TestParseArgsRequiresScript(t *testing.T) {
	handler := NewHandler().ParseArgs(nil)
	if handler.err == nil || !strings.Contains(handler.err.Error(), "requires exactly one --script") {
		t.Fatalf("expected missing --script error, got %v", handler.err)
	}
}
