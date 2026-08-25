package utils

import (
	"fmt"
	"path/filepath"
)

func AbsCleanPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}
	return filepath.Clean(absolute), nil
}
