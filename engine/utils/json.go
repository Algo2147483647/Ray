package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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

func DecodeStrictJSON(data []byte, kind string, target interface{}) error {
	fields, err := JSONFieldNames(target)
	if err != nil {
		return err
	}
	if err := RejectUnknownJSONFields(data, kind, fields...); err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// JSONFieldNames returns the serialized field names declared by struct tags.
// It is the single source for strict-object decoding; callers only add fields
// that belong to another member of a discriminated union.
func JSONFieldNames(values ...interface{}) ([]string, error) {
	seen := make(map[string]struct{})
	for _, value := range values {
		typeOf := reflect.TypeOf(value)
		for typeOf != nil && (typeOf.Kind() == reflect.Pointer || typeOf.Kind() == reflect.Interface) {
			typeOf = typeOf.Elem()
		}
		if typeOf == nil || typeOf.Kind() != reflect.Struct {
			return nil, fmt.Errorf("strict JSON target must be a struct, got %T", value)
		}
		collectJSONFields(typeOf, seen)
	}
	fields := make([]string, 0, len(seen))
	for field := range seen {
		fields = append(fields, field)
	}
	return fields, nil
}

func RejectUnknownJSONFieldsFor(data []byte, kind string, values ...interface{}) error {
	fields, err := JSONFieldNames(values...)
	if err != nil {
		return err
	}
	return RejectUnknownJSONFields(data, kind, fields...)
}

func collectJSONFields(typeOf reflect.Type, seen map[string]struct{}) {
	for index := 0; index < typeOf.NumField(); index++ {
		field := typeOf.Field(index)
		tag := field.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "-" {
			continue
		}
		if name == "" {
			if field.Anonymous {
				nested := field.Type
				for nested.Kind() == reflect.Pointer {
					nested = nested.Elem()
				}
				if nested.Kind() == reflect.Struct {
					collectJSONFields(nested, seen)
				}
			}
			continue
		}
		seen[name] = struct{}{}
	}
}
