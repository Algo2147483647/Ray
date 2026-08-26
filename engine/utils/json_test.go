package utils

import (
	"strings"
	"testing"
)

func TestDecodeStrictJSONRejectsUnknownField(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	err := DecodeStrictJSON([]byte(`{"name":"scene","extra":true}`), "test object", &target)
	if err == nil || !strings.Contains(err.Error(), `unsupported test object field "extra"`) {
		t.Fatalf("expected an unknown-field error, got %v", err)
	}
}

func TestDecodeStrictJSONRejectsNonObject(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	if err := DecodeStrictJSON([]byte(`[]`), "test object", &target); err == nil {
		t.Fatal("array must not decode as an object")
	}
}

func TestRejectUnknownJSONFieldsForCombinesDiscriminatedStructTags(t *testing.T) {
	header := struct {
		Type string `json:"type"`
	}{}
	payload := struct {
		Value int `json:"value,omitempty"`
	}{}
	if err := RejectUnknownJSONFieldsFor([]byte(`{"type":"number","value":3}`), "variant", &header, &payload); err != nil {
		t.Fatalf("tag-derived union fields rejected: %v", err)
	}
	err := RejectUnknownJSONFieldsFor([]byte(`{"type":"number","other":3}`), "variant", &header, &payload)
	if err == nil || !strings.Contains(err.Error(), `unsupported variant field "other"`) {
		t.Fatalf("expected unknown union field error, got %v", err)
	}
}
