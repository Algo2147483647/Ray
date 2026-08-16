package camera

import (
	"encoding/binary"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/Algo2147483647/ray/engine/model/optics"
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

func TestFilmToImageConvertsACEScgColorSpace(t *testing.T) {
	film := NewFilm(1, 1)
	film.ColorSpace = FilmColorSpaceACEScg
	r, g, b := XYZToFilmColorSpace(0.95047, 1, 1.08883, FilmColorSpaceACEScg)
	film.Data[0].Data[0] = r
	film.Data[1].Data[0] = g
	film.Data[2].Data[0] = b

	img := film.ToImageWithOptions(ImageOptions{
		Exposure:    1,
		ToneMapping: ToneMappingLinear,
		Gamma:       1,
	})

	got := img.RGBAAt(0, 0)
	if got.R < 250 || got.G < 250 || got.B < 250 {
		t.Fatalf("expected ACEScg D65-like white to convert near display white, got %+v", got)
	}
}

func TestFilmConvertsSpectralBinsToFilmColorSpace(t *testing.T) {
	film := NewFilm(1, 1)
	film.ColorSpace = FilmColorSpaceXYZ
	film.InitSpectralBins(1, 549.5, 550.5)
	film.RecordSpectralSample(0, 550, 1)

	film.ConvertSpectralBinsToFilmColorSpace()
	want := optics.SpectralRadianceToXYZ(550, 1)

	for ch := 0; ch < 3; ch++ {
		if got := film.Data[ch].Data[0]; math.Abs(got-want[ch]) > 1e-12 {
			t.Fatalf("unexpected spectral conversion channel %d: got %f want %f", ch, got, want[ch])
		}
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

func TestFilmFileRoundTripsACEScgAndSpectralBins(t *testing.T) {
	film := NewFilm(1, 1)
	film.ColorSpace = FilmColorSpaceACEScg
	film.InitSpectralBins(2, 400, 800)
	film.RecordSpectralSample(0, 450, 1.25)
	film.RecordSpectralSample(0, 650, 2.5)

	filename := filepath.Join(t.TempDir(), "film.bin")
	if err := film.SaveToFile(filename); err != nil {
		t.Fatalf("save film: %v", err)
	}

	loaded := NewFilm(1, 1)
	if err := loaded.LoadFromFile(filename); err != nil {
		t.Fatalf("load film: %v", err)
	}
	if loaded.ColorSpace != FilmColorSpaceACEScg {
		t.Fatalf("expected ACEScg working space to round-trip, got %q", loaded.ColorSpace)
	}
	if !loaded.HasSpectralBins() || len(loaded.SpectralBins) != 2 {
		t.Fatalf("expected spectral bins to round-trip, got %+v", loaded.SpectralBins)
	}
	if got := loaded.SpectralBins[0].Data[0]; math.Abs(got-1.25) > 1e-12 {
		t.Fatalf("unexpected first spectral bin: %f", got)
	}
	if got := loaded.SpectralBins[1].Data[0]; math.Abs(got-2.5) > 1e-12 {
		t.Fatalf("unexpected second spectral bin: %f", got)
	}
}

func TestFilmFileRoundTripsAcrossCodecBlocks(t *testing.T) {
	const width = filmFloatChunkBytes/8 + 17
	film := NewFilm(width, 1)
	film.Samples = 123
	film.ColorSpace = FilmColorSpaceXYZ
	film.InitSpectralBins(2, 400, 800)
	for i := range film.Data[0].Data {
		for channel := range film.Data {
			film.Data[channel].Data[i] = float64(i*3+channel) / 7
		}
		for bin := range film.SpectralBins {
			film.SpectralBins[bin].Data[i] = float64(i+bin) / 11
		}
	}

	filename := filepath.Join(t.TempDir(), "chunked-film.bin")
	if err := film.SaveToFile(filename); err != nil {
		t.Fatalf("save film: %v", err)
	}
	loaded := NewFilm()
	if err := loaded.LoadFromFile(filename); err != nil {
		t.Fatalf("load film: %v", err)
	}

	for _, index := range []int{0, width/2 - 1, width / 2, width - 1} {
		for channel := range film.Data {
			if got, want := loaded.Data[channel].Data[index], film.Data[channel].Data[index]; got != want {
				t.Fatalf("color channel %d index %d = %v, want %v", channel, index, got, want)
			}
		}
		for bin := range film.SpectralBins {
			if got, want := loaded.SpectralBins[bin].Data[index], film.SpectralBins[bin].Data[index]; got != want {
				t.Fatalf("spectral bin %d index %d = %v, want %v", bin, index, got, want)
			}
		}
	}
}

func TestFilmFileRejectsLegacyHeaderlessFormat(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "legacy.bin")
	if err := os.WriteFile(filename, make([]byte, 64), 0o644); err != nil {
		t.Fatalf("write legacy fixture: %v", err)
	}
	if err := NewFilm().LoadFromFile(filename); err == nil {
		t.Fatal("expected legacy Film format to be rejected")
	}
}

func TestFilmFileWritesCurrentMagicAndVersion(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "header.bin")
	if err := NewFilm(1, 1).SaveToFile(filename); err != nil {
		t.Fatalf("save film: %v", err)
	}
	header, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read film: %v", err)
	}
	if len(header) < 12 || string(header[:8]) != string(filmFileMagic[:]) {
		t.Fatalf("unexpected Film magic: %x", header[:min(len(header), 8)])
	}
	if got := binary.LittleEndian.Uint32(header[8:12]); got != filmFileVersion {
		t.Fatalf("Film version = %d, want %d", got, filmFileVersion)
	}
}

func TestFilmFileRejectsTrailingPayload(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "trailing.bin")
	film := NewFilm(1, 1)
	if err := film.SaveToFile(filename); err != nil {
		t.Fatalf("save film: %v", err)
	}
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open film: %v", err)
	}
	if _, err := file.Write([]byte{0}); err != nil {
		file.Close()
		t.Fatalf("append payload: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close film: %v", err)
	}
	if err := NewFilm().LoadFromFile(filename); err == nil {
		t.Fatal("expected trailing Film payload to be rejected")
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
