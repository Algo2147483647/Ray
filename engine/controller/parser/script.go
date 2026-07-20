package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ReadScriptFile(path string) (*Script, error) {
	absolute, err := filepathAbs(path)
	if err != nil {
		return nil, err
	}
	return readScriptFileRaw(absolute)
}

func readScriptFileRaw(path string) (*Script, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open script %q: %w", path, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read script %q: %w", path, err)
	}

	var script Script
	if err := json.Unmarshal(data, &script); err != nil {
		return nil, fmt.Errorf("parse script %q: %w", path, err)
	}

	return &script, nil
}

func filepathAbs(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve script path %q: %w", path, err)
	}
	return filepath.Clean(absolute), nil
}
