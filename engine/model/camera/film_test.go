package camera

import (
	"image/color"
	"path/filepath"
	"testing"
)

func TestFilmToImageDefaultMatchesLinearClamp(t *testing.T) {
	film := NewFilm(2, 1)
	film.Data[0].Data[0] = 0.5
	film.Data[1].Data[0] = 1.5
	film.Data[2].Data[0] = -1
	film.Data[0].Data[1] = 2
	film.Data[1].Data[1] = 0.25
	film.Data[2].Data[1] = 0

	img := film.ToImage()
	if got := img.RGBAAt(0, 0); got != (color.RGBA{128, 255, 0, 255}) {
		t.Fatalf("unexpected first pixel: %+v", got)
	}
	if got := img.RGBAAt(1, 0); got != (color.RGBA{255, 64, 0, 255}) {
		t.Fatalf("unexpected second pixel: %+v", got)
	}
}

func TestFilmToImageWithReinhardExposureAndGamma(t *testing.T) {
	film := NewFilm(1, 1)
	film.Data[0].Data[0] = 4
	film.Data[1].Data[0] = 1
	film.Data[2].Data[0] = 0.25

	img := film.ToImageWithOptions(ImageOptions{
		Exposure:    0.5,
		ToneMapping: ToneMappingReinhard,
		Gamma:       2,
	})

	if got := img.RGBAAt(0, 0); got != (color.RGBA{208, 147, 85, 255}) {
		t.Fatalf("unexpected tone-mapped pixel: %+v", got)
	}
}

func TestFilmToImageWithACESCompressesHighlights(t *testing.T) {
	film := NewFilm(1, 1)
	film.Data[0].Data[0] = 8
	film.Data[1].Data[0] = 1
	film.Data[2].Data[0] = 0.1

	img := film.ToImageWithOptions(ImageOptions{
		Exposure:    1,
		ToneMapping: ToneMappingACES,
		Gamma:       1,
	})

	got := img.RGBAAt(0, 0)
	if got.R != 255 {
		t.Fatalf("expected ACES highlight to approach display white, got %+v", got)
	}
	if got.G <= got.B {
		t.Fatalf("expected green channel to remain above blue, got %+v", got)
	}
}

func TestFilmToImageConvertsXYZColorSpace(t *testing.T) {
	film := NewFilm(1, 1)
	film.ColorSpace = FilmColorSpaceXYZ
	film.Data[0].Data[0] = 0.95047
	film.Data[1].Data[0] = 1
	film.Data[2].Data[0] = 1.08883

	img := film.ToImageWithOptions(ImageOptions{
		Exposure:    1,
		ToneMapping: ToneMappingLinear,
		Gamma:       1,
	})

	got := img.RGBAAt(0, 0)
	if got.R < 250 || got.G < 250 || got.B < 250 {
		t.Fatalf("expected D65-like XYZ white to convert near display white, got %+v", got)
	}
}

func TestFilmFileRoundTripsColorSpace(t *testing.T) {
	film := NewFilm(1, 1)
	film.ColorSpace = FilmColorSpaceXYZ
	film.Data[0].Data[0] = 0.95047
	film.Data[1].Data[0] = 1
	film.Data[2].Data[0] = 1.08883

	filename := filepath.Join(t.TempDir(), "film.bin")
	if err := film.SaveToFile(filename); err != nil {
		t.Fatalf("save film: %v", err)
	}

	loaded := NewFilm(1, 1)
	if err := loaded.LoadFromFile(filename); err != nil {
		t.Fatalf("load film: %v", err)
	}
	if loaded.ColorSpace != FilmColorSpaceXYZ {
		t.Fatalf("expected working space to round-trip as XYZ, got %q", loaded.ColorSpace)
	}
}

func TestFilmRecordsSpectralDiagnostics(t *testing.T) {
	film := NewFilm(2, 1)
	film.InitSpectralBins(4, 400, 800)

	film.RecordSpectralSample(0, 450, 1.5)
	film.RecordSpectralSample(1, 750, 2.5)

	if !film.HasSpectralBins() {
		t.Fatal("expected spectral bins to be enabled")
	}
	first := film.SpectralBinIndex(450)
	last := film.SpectralBinIndex(750)
	if first == last || first < 0 || last < 0 {
		t.Fatalf("expected wavelengths to land in different bins, got %d and %d", first, last)
	}
	if film.SpectralBins[first].Data[0] != 1.5 {
		t.Fatalf("unexpected first spectral bin value: %f", film.SpectralBins[first].Data[0])
	}
	if film.SpectralBins[last].Data[1] != 2.5 {
		t.Fatalf("unexpected last spectral bin value: %f", film.SpectralBins[last].Data[1])
	}
}
