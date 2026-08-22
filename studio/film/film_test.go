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

func TestSpectralTanhUsesOneScaleForEveryBin(t *testing.T) {
	film := spectralFilm([]int{2, 1}, 3, 1, 0)
	values := [][2]float64{{1, 2}, {2, 4}, {3, 6}}
	for bin := range film.SpectralBins {
		copy(film.SpectralBins[bin].Data, values[bin][:])
	}

	x, y, z, brightness, err := spectralXYZAndBrightnessAt(film, 0, 1)
	if err != nil {
		t.Fatalf("spectral pixel: %v", err)
	}
	if x <= 0 || y <= 0 || z <= 0 {
		t.Fatalf("expected positive XYZ, got %g %g %g", x, y, z)
	}
	assertClose(t, brightness, 6)
	scale := spectralTanhScale(brightness, 1, 0.5)
	assertClose(t, brightness*scale, math.Tanh(3))
	for bin := 1; bin < len(film.SpectralBins); bin++ {
		gotRatio := film.SpectralBins[bin].Data[0] * scale / (film.SpectralBins[0].Data[0] * scale)
		wantRatio := film.SpectralBins[bin].Data[0] / film.SpectralBins[0].Data[0]
		assertClose(t, gotRatio, wantRatio)
	}
}

func TestSpectralTanhRejectsNonPhysicalSpectrum(t *testing.T) {
	film := spectralFilm([]int{1, 1}, 3, 1, 1)
	film.SpectralBins[1].Data[0] = -0.01
	if _, err := ToImage(film, ImageOptions{ToneMapping: ToneMappingSpectralTanh}); err == nil {
		t.Fatal("expected a negative spectral channel to be rejected")
	}
}

func TestSpectralTanhFitsRGBWithOneSharedScale(t *testing.T) {
	r, g, b := fitRGBWithoutChannelClipping(2, 1, 0.5)
	assertClose(t, r, 1)
	assertClose(t, g, 0.5)
	assertClose(t, b, 0.25)
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
