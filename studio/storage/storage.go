package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Algo2147483647/ray/studio/schema"
)

func WriteIntermediateScript(script *schema.IntermediateScript, source []string) (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	outputDir := filepath.Join(root, "outputs", "intermediate")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", fmt.Errorf("create intermediate directory: %w", err)
	}

	name := intermediateName(source)
	outputPath := filepath.Join(outputDir, name)
	data, err := json.MarshalIndent(script, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode intermediate script: %w", err)
	}
	if err := os.WriteFile(outputPath, append(data, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write intermediate script: %w", err)
	}
	return outputPath, nil
}

func intermediateName(source []string) string {
	base := "merged"
	if len(source) == 1 {
		trimmed := strings.TrimSuffix(filepath.Base(source[0]), filepath.Ext(source[0]))
		if trimmed != "" {
			base = trimmed
		}
	}
	hash := fnv.New32a()
	for _, path := range source {
		_, _ = hash.Write([]byte(path))
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("%s.studio.%08x.json", sanitizeFilename(base), hash.Sum32())
}

func RepoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("resolve studio source path")
	}
	return filepath.Abs(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func sanitizeFilename(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteByte('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "script"
	}
	return result
}
