package utils

import (
	"encoding/json"
	"fmt"
)

func RejectUnknownJSONFields(data []byte, kind string, allowed ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("%s must be an object", kind)
	}
	return RejectUnknownMapFields(fields, kind, allowed...)
}

func RejectUnknownMapFields[T any](fields map[string]T, kind string, allowed ...string) error {
	known := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		known[field] = struct{}{}
	}
	for field := range fields {
		if _, ok := known[field]; !ok {
			return fmt.Errorf("unsupported %s field %q", kind, field)
		}
	}
	return nil
}

func DecodeStrictJSON(data []byte, kind string, target interface{}, allowed ...string) error {
	if err := RejectUnknownJSONFields(data, kind, allowed...); err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
