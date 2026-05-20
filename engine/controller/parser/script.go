package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func ReadScriptFile(filepath string) (*Script, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("open script %q: %w", filepath, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", filepath, err)
	}

	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("parse script %q: %w", filepath, err)
	}

	return &script, nil
}
