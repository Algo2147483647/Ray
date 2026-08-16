package camera

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestSpectralFilmRecordsSamples(t *testing.T) {
	film := NewFilm(2, 1)
	film.InitSpectralBins(8, 380, 750)
	film.RecordSpectralSample(0, 400, 1.5)
	film.RecordSpectralSample(1, math.Nextafter(750, 380), 2.5)
	if got := film.SpectralBins[film.SpectralBinIndex(400)].Data[0]; got != 1.5 {
		t.Fatalf("first spectral value = %g, want 1.5", got)
	}
	if got := film.SpectralBins[film.SpectralBinIndex(math.Nextafter(750, 380))].Data[1]; got != 2.5 {
		t.Fatalf("last spectral value = %g, want 2.5", got)
	}
}

func TestSpectralFilmV3RoundTrip(t *testing.T) {
	film := NewFilm(3, 2)
	film.Samples = 17
	film.InitSpectralBins(5, 380, 750)
	for bin := range film.SpectralBins {
		for pixel := range film.SpectralBins[bin].Data {
			film.SpectralBins[bin].Data[pixel] = float64(bin*100+pixel) / 7
		}
	}
	path := filepath.Join(t.TempDir(), "film.bin")
	if err := film.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	loaded := NewFilm()
	if err := loaded.LoadFromFile(path); err != nil {
		t.Fatal(err)
	}
	if loaded.Samples != film.Samples || len(loaded.Shape) != 2 || loaded.Shape[0] != 3 || loaded.Shape[1] != 2 {
		t.Fatalf("round-trip metadata mismatch: %+v", loaded)
	}
	for bin := range film.SpectralBins {
		for pixel, want := range film.SpectralBins[bin].Data {
			if got := loaded.SpectralBins[bin].Data[pixel]; got != want {
				t.Fatalf("bin %d pixel %d = %g, want %g", bin, pixel, got, want)
			}
		}
	}
}

func TestSpectralFilmRejectsOldVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.bin")
	data := append([]byte(nil), filmFileMagic[:]...)
	version := make([]byte, 4)
	binary.LittleEndian.PutUint32(version, 2)
	data = append(data, version...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := NewFilm().LoadFromFile(path); err == nil {
		t.Fatal("expected v2 Film to be rejected")
	}
}

func TestSpectralFilmChunkedPayloadRoundTrip(t *testing.T) {
	width := filmFloatChunkBytes/8 + 17
	film := NewFilm(width, 1)
	film.InitSpectralBins(1, 380, 750)
	for i := range film.SpectralBins[0].Data {
		film.SpectralBins[0].Data[i] = float64(i) * 0.125
	}
	path := filepath.Join(t.TempDir(), "large.bin")
	if err := film.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	loaded := NewFilm()
	if err := loaded.LoadFromFile(path); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, width/2 - 1, width / 2, width - 1} {
		if got, want := loaded.SpectralBins[0].Data[index], film.SpectralBins[0].Data[index]; got != want {
			t.Fatalf("pixel %d = %g, want %g", index, got, want)
		}
	}
}
