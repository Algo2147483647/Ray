package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	studiofilm "github.com/Algo2147483647/ray/studio/film"
	"github.com/Algo2147483647/ray/studio/schema"
)

func TestResolveEngineProcessUsesConfiguredBinaryWithoutRepoLayout(t *testing.T) {
	engineBin := filepath.Join(t.TempDir(), "custom-engine")
	if err := os.WriteFile(engineBin, []byte("test executable placeholder"), 0o700); err != nil {
		t.Fatalf("write Engine placeholder: %v", err)
	}

	process, err := resolveEngineProcess(studioConfig{engineBin: engineBin})
	if err != nil {
		t.Fatalf("resolve configured Engine executable: %v", err)
	}
	if process.executable != engineBin {
		t.Fatalf("Engine executable = %q, want %q", process.executable, engineBin)
	}
	if process.workingDir != "" {
		t.Fatalf("configured Engine unexpectedly depends on working directory %q", process.workingDir)
	}
}

func TestStudioIntermediateContractRunsInEngineProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Engine process contract test in short mode")
	}

	engineDir, err := filepath.Abs(filepath.Join("..", "engine"))
	if err != nil {
		t.Fatalf("resolve Engine directory: %v", err)
	}
	binaryName := "ray-engine"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	engineBin := filepath.Join(t.TempDir(), binaryName)
	build := exec.Command("go", "build", "-o", engineBin, ".")
	build.Dir = engineDir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build independent Engine executable: %v\n%s", err, output)
	}

	dir := t.TempDir()
	filmPath := filepath.Join(dir, "contract.bin")
	adapted, err := adaptTestScript(&schema.StudioScript{
		Dimension: 3,
		Cameras:   []schema.StudioCameraScript{{ID: "main", Type: "3d"}},
		Films: []schema.StudioFilmScript{{
			ID: "main-film", CameraID: "main", Shape: []int{1, 1},
			SpectralBinCount: 4, OutputFilm: filmPath,
		}},
		Render: schema.StudioRenderScript{FilmID: "main-film", Samples: 1, ThreadNum: 1},
	}, []string{"process-contract.json"}, 3)
	if err != nil {
		t.Fatalf("adapt Studio contract script: %v", err)
	}
	data, err := json.Marshal(adapted)
	if err != nil {
		t.Fatalf("marshal Studio contract script: %v", err)
	}
	scriptPath := filepath.Join(dir, "intermediate.json")
	if err := os.WriteFile(scriptPath, data, 0o600); err != nil {
		t.Fatalf("write Studio contract script: %v", err)
	}

	code, err := (engineProcess{executable: engineBin, workingDir: engineDir}).run([]string{"--script", scriptPath})
	if err != nil {
		t.Fatalf("start Engine process: %v", err)
	}
	if code != 0 {
		t.Fatalf("Engine process exited with code %d", code)
	}

	film := &studiofilm.Film{}
	if err := film.LoadFromFile(filmPath); err != nil {
		t.Fatalf("Studio could not read Engine process output: %v", err)
	}
	if len(film.Shape) != 2 || film.Shape[0] != 1 || film.Shape[1] != 1 {
		t.Fatalf("Engine Film shape = %v, want [1 1]", film.Shape)
	}
	if len(film.SpectralBins) != 4 || film.Samples != 1 {
		t.Fatalf("Engine Film contract mismatch: bins=%d samples=%d", len(film.SpectralBins), film.Samples)
	}
}
