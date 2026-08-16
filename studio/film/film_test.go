package film

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

func spectralFilm(shape []int, bins int, samples int64, value float64) *modelcamera.Film {
	film := modelcamera.NewFilm(shape...)
	film.InitSpectralBins(bins, 380, 750)
	film.Samples = samples
	for bin := range film.SpectralBins {
		for pixel := range film.SpectralBins[bin].Data {
			film.SpectralBins[bin].Data[pixel] = value
		}
	}
	return film
}

func TestMergeFilmFilesWritesWeightedSpectralMerge(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.bin")
	updatePath := filepath.Join(dir, "update.bin")
	outputPath := filepath.Join(dir, "merged.bin")

	base := spectralFilm([]int{1, 1}, 4, 2, 0.25)
	update := spectralFilm([]int{1, 1}, 4, 6, 0.75)
	if err := base.SaveToFile(basePath); err != nil {
		t.Fatalf("save base film: %v", err)
	}
	if err := update.SaveToFile(updatePath); err != nil {
		t.Fatalf("save update film: %v", err)
	}

	if err := MergeFilmFiles(basePath, updatePath, outputPath); err != nil {
		t.Fatalf("merge films: %v", err)
	}
	merged, err := LoadFilm(outputPath)
	if err != nil {
		t.Fatalf("load merged film: %v", err)
	}
	if merged.Samples != 8 {
		t.Fatalf("samples = %d, want 8", merged.Samples)
	}
	for bin := range merged.SpectralBins {
		assertClose(t, merged.SpectralBins[bin].Data[0], 0.625)
	}
}

func TestMergeFilmFilesWithPixelWindowsLeavesOutsidePixelsUntouched(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.bin")
	updatePath := filepath.Join(dir, "update.bin")
	outputPath := filepath.Join(dir, "merged.bin")

	base := spectralFilm([]int{4, 4}, 3, 2, 0.2)
	update := spectralFilm([]int{4, 4}, 3, 6, 1)
	if err := base.SaveToFile(basePath); err != nil {
		t.Fatalf("save base film: %v", err)
	}
	if err := update.SaveToFile(updatePath); err != nil {
		t.Fatalf("save update film: %v", err)
	}

	windows := []modelcamera.PixelWindow{{Min: []int{1, 1}, Max: []int{3, 3}}}
	if err := MergeFilmFilesWithPixelWindows(basePath, updatePath, outputPath, windows); err != nil {
		t.Fatalf("merge films with pixel windows: %v", err)
	}
	merged, err := LoadFilm(outputPath)
	if err != nil {
		t.Fatalf("load merged film: %v", err)
	}
	if merged.Samples != 8 {
		t.Fatalf("samples = %d, want 8", merged.Samples)
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			expected := 0.2
			if x >= 1 && x < 3 && y >= 1 && y < 3 {
				expected = 0.8
			}
			assertClose(t, merged.SpectralBins[0].Data[y*4+x], expected)
		}
	}
}

func TestSaveFilmImageFromFilmDoesNotRequireFilmFile(t *testing.T) {
	film := spectralFilm([]int{1, 1}, 64, 1, 1.0/64)
	imagePath := filepath.Join(t.TempDir(), "direct.png")

	if err := SaveFilmImageFromFilm(film, imagePath, ImageOptions{}); err != nil {
		t.Fatalf("save image from in-memory Film: %v", err)
	}
	if info, err := os.Stat(imagePath); err != nil || info.Size() == 0 {
		t.Fatalf("expected non-empty image at %q: info=%v err=%v", imagePath, info, err)
	}
}

func TestToImageWorkingSpacesPreserveTheSameXYZColor(t *testing.T) {
	film := spectralFilm([]int{1, 1}, 128, 1, 1.0/128)
	var reference [3]uint8
	for _, space := range []ColorSpace{ColorSpaceLinearSRGB, ColorSpaceXYZ, ColorSpaceACEScg} {
		img, err := ToImage(film, ImageOptions{ColorSpace: space})
		if err != nil {
			t.Fatalf("ToImage(%s): %v", space, err)
		}
		pixel := img.RGBAAt(0, 0)
		if space == ColorSpaceLinearSRGB {
			reference = [3]uint8{pixel.R, pixel.G, pixel.B}
			continue
		}
		for channel, got := range []uint8{pixel.R, pixel.G, pixel.B} {
			if byteDistance(got, reference[channel]) > 1 {
				t.Fatalf("working space %s changed output color: got %v, reference %v", space, pixel, reference)
			}
		}
	}
}

func TestToImageRejectsOutputSettingsUnknownToStudio(t *testing.T) {
	film := spectralFilm([]int{1, 1}, 8, 1, 1.0/8)
	if _, err := ToImage(film, ImageOptions{ToneMapping: "legacy"}); err == nil {
		t.Fatal("expected unsupported tone mapping to fail")
	}
	if _, err := ToImage(film, ImageOptions{ColorSpace: "legacy"}); err == nil {
		t.Fatal("expected unsupported working color space to fail")
	}
}

func byteDistance(a, b uint8) uint8 {
	if a >= b {
		return a - b
	}
	return b - a
}

func assertClose(t *testing.T, got, expected float64) {
	t.Helper()
	if math.Abs(got-expected) > 1e-10 {
		t.Fatalf("expected %f, got %f", expected, got)
	}
}
