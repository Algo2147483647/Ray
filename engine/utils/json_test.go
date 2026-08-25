package utils

import (
	"strings"
	"testing"
)

func TestDecodeStrictJSONRejectsUnknownField(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	err := DecodeStrictJSON([]byte(`{"name":"scene","extra":true}`), "test object", &target, "name")
	if err == nil || !strings.Contains(err.Error(), `unsupported test object field "extra"`) {
		t.Fatalf("expected an unknown-field error, got %v", err)
	}
}

func TestDecodeStrictJSONRejectsNonObject(t *testing.T) {
	var target map[string]interface{}
	if err := DecodeStrictJSON([]byte(`[]`), "test object", &target); err == nil {
		t.Fatal("array must not decode as an object")
	}
}
