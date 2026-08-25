package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const engineBinaryEnv = "RAY_ENGINE_BIN"

type engineProcess struct {
	executable string
	workingDir string
}

func resolveEngineProcess(config studioConfig) (engineProcess, error) {
	requested := strings.TrimSpace(config.engineBin)
	if requested == "" {
		requested = strings.TrimSpace(os.Getenv(engineBinaryEnv))
	}
	if requested != "" {
		executable, err := resolveEngineExecutable(requested)
		if err != nil {
			return engineProcess{}, err
		}
		return engineProcess{executable: executable}, nil
	}

	for _, name := range []string{"ray-engine", "engine"} {
		if executable, err := exec.LookPath(name); err == nil {
			return engineProcess{executable: executable}, nil
		}
	}
	if studioExecutable, err := os.Executable(); err == nil {
		for _, name := range []string{"ray-engine", "ray-engine.exe", "engine", "engine.exe"} {
			candidate := filepath.Join(filepath.Dir(studioExecutable), name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return engineProcess{executable: candidate}, nil
			}
		}
	}
	return engineProcess{}, fmt.Errorf("Engine executable not found; pass --engine-bin, set %s, add ray-engine to PATH, or install it beside Studio", engineBinaryEnv)
}

func resolveEngineExecutable(value string) (string, error) {
	if strings.ContainsAny(value, `/\\`) || filepath.IsAbs(value) {
		absolute, err := filepath.Abs(value)
		if err != nil {
			return "", fmt.Errorf("resolve Engine executable %q: %w", value, err)
		}
		info, err := os.Stat(absolute)
		if err != nil {
			return "", fmt.Errorf("Engine executable %q: %w", absolute, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("Engine executable %q is a directory", absolute)
		}
		return absolute, nil
	}
	executable, err := exec.LookPath(value)
	if err != nil {
		return "", fmt.Errorf("find Engine executable %q: %w", value, err)
	}
	return executable, nil
}

func (process engineProcess) run(args []string) (int, error) {
	command := exec.Command(process.executable, args...)
	if process.workingDir != "" {
		command.Dir = process.workingDir
	}
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Run()
	if err == nil {
		return 0, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode(), nil
	}
	return 0, fmt.Errorf("start Engine process %q: %w", process.executable, err)
}
