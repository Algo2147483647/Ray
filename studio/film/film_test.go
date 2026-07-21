package film

import (
	"math"
	"path/filepath"
	"testing"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

func TestMergeFilmFilesWritesWeightedMerge(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.bin")
	updatePath := filepath.Join(dir, "update.bin")
	outputPath := filepath.Join(dir, "merged.bin")

	base := modelcamera.NewFilm(1, 1)
	base.Samples = 2
	base.Data[0].Data[0] = 0.25
	base.Data[1].Data[0] = 0.5
	base.Data[2].Data[0] = 0.75
	if err := base.SaveToFile(basePath); err != nil {
		t.Fatalf("save base film: %v", err)
	}

	update := modelcamera.NewFilm(1, 1)
	update.Samples = 6
	update.Data[0].Data[0] = 0.75
	update.Data[1].Data[0] = 0.25
	update.Data[2].Data[0] = 0.5
	if err := update.SaveToFile(updatePath); err != nil {
		t.Fatalf("save update film: %v", err)
	}

	if err := MergeFilmFiles(basePath, updatePath, outputPath); err != nil {
		t.Fatalf("merge films: %v", err)
	}

	merged := modelcamera.NewFilm()
	if err := merged.LoadFromFile(outputPath); err != nil {
		t.Fatalf("load merged film: %v", err)
	}
	if merged.Samples != 8 {
		t.Fatalf("expected 8 merged samples, got %d", merged.Samples)
	}
	assertClose(t, merged.Data[0].Data[0], 0.625)
	assertClose(t, merged.Data[1].Data[0], 0.3125)
	assertClose(t, merged.Data[2].Data[0], 0.5625)
}

func assertClose(t *testing.T, got, expected float64) {
	t.Helper()
	if math.Abs(got-expected) > 1e-10 {
		t.Fatalf("expected %f, got %f", expected, got)
	}
}
