package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Algo2147483647/ray/engine/utils"
)

func ReadScriptFile(path string) (*Script, error) {
	absolute, err := utils.AbsCleanPath(path)
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
