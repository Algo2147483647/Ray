package camera

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"slices"

	"github.com/Algo2147483647/ray/engine/maths"
	"github.com/Algo2147483647/ray/engine/utils/binaryio"
)

var filmFileMagic = [8]byte{'R', 'A', 'Y', 'F', 'I', 'L', 'M', 0}

const (
	filmFileVersion     uint32 = 3
	filmFloatChunkBytes        = 1 << 20
	maxFilmRank         uint32 = 16
	maxFilmSpectralBins uint32 = 4096
)

var filmByteOrder = binary.LittleEndian

type filmFileHeader struct {
	Magic   [8]byte
	Version uint32
	Samples int64
	Rank    uint32
}

type filmSpectrumHeader struct {
	BinCount uint32
	MinNM    float64
	MaxNM    float64
}

type filmFileMetadata struct {
	Samples      int64
	Shape        []int
	BinCount     int
	MinNM        float64
	MaxNM        float64
	ElementCount int
}

func (f *Film) SaveToFile(filename string) error {
	metadata, err := metadataForFilm(f)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create Film %q: %w", filename, err)
	}
	if err := writeFilm(file, f, metadata); err != nil {
		_ = file.Close()
		return fmt.Errorf("write Film %q: %w", filename, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close Film %q: %w", filename, err)
	}
	return nil
}

func (f *Film) LoadFromFile(filename string) error {
	if f == nil {
		return fmt.Errorf("cannot load into a nil Film")
	}

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open Film %q: %w", filename, err)
	}
	defer file.Close()

	loaded, err := readFilm(file)
	if err != nil {
		return fmt.Errorf("read Film %q: %w", filename, err)
	}

	// Commit only after the complete file has been decoded successfully.
	f.Shape = loaded.Shape
	f.Samples = loaded.Samples
	f.SpectralBinCount = loaded.SpectralBinCount
	f.SpectralBins = loaded.SpectralBins
	f.SpectralMinNM = loaded.SpectralMinNM
	f.SpectralMaxNM = loaded.SpectralMaxNM
	return nil
}

func writeFilm(w io.Writer, film *Film, metadata filmFileMetadata) error {
	header := filmFileHeader{
		Magic:   filmFileMagic,
		Version: filmFileVersion,
		Samples: metadata.Samples,
		Rank:    uint32(len(metadata.Shape)),
	}
	if err := binary.Write(w, filmByteOrder, header); err != nil {
		return fmt.Errorf("header: %w", err)
	}

	dimensions := make([]uint64, len(metadata.Shape))
	for axis, extent := range metadata.Shape {
		dimensions[axis] = uint64(extent)
	}
	if err := binary.Write(w, filmByteOrder, dimensions); err != nil {
		return fmt.Errorf("shape: %w", err)
	}

	spectrum := filmSpectrumHeader{
		BinCount: uint32(metadata.BinCount),
		MinNM:    metadata.MinNM,
		MaxNM:    metadata.MaxNM,
	}
	if err := binary.Write(w, filmByteOrder, spectrum); err != nil {
		return fmt.Errorf("spectrum header: %w", err)
	}

	buffer := make([]byte, filmFloatChunkBytes)
	for bin := range film.SpectralBins {
		if err := binaryio.WriteFloat64s(w, filmByteOrder, film.SpectralBins[bin].Data, buffer); err != nil {
			return fmt.Errorf("spectral plane %d: %w", bin, err)
		}
	}
	return nil
}

func readFilm(r io.Reader) (*Film, error) {
	metadata, err := readFilmMetadata(r)
	if err != nil {
		return nil, err
	}

	film := &Film{
		Shape:            metadata.Shape,
		Samples:          metadata.Samples,
		SpectralBinCount: metadata.BinCount,
		SpectralBins:     make([]maths.Tensor[float64], metadata.BinCount),
		SpectralMinNM:    metadata.MinNM,
		SpectralMaxNM:    metadata.MaxNM,
	}
	buffer := make([]byte, filmFloatChunkBytes)
	for bin := range film.SpectralBins {
		film.SpectralBins[bin] = *maths.NewTensor[float64](metadata.Shape)
		if err := binaryio.ReadFloat64s(r, filmByteOrder, film.SpectralBins[bin].Data, buffer); err != nil {
			return nil, fmt.Errorf("spectral plane %d: %w", bin, err)
		}
	}
	if err := binaryio.RequireEOF(r); err != nil {
		return nil, err
	}
	return film, nil
}

func readFilmMetadata(r io.Reader) (filmFileMetadata, error) {
	var header filmFileHeader
	if err := binary.Read(r, filmByteOrder, &header); err != nil {
		return filmFileMetadata{}, fmt.Errorf("header: %w", err)
	}
	if header.Magic != filmFileMagic {
		return filmFileMetadata{}, fmt.Errorf("invalid magic")
	}
	if header.Version != filmFileVersion {
		return filmFileMetadata{}, fmt.Errorf("unsupported version %d; expected %d", header.Version, filmFileVersion)
	}
	if header.Rank == 0 || header.Rank > maxFilmRank {
		return filmFileMetadata{}, fmt.Errorf("invalid rank %d", header.Rank)
	}

	dimensions := make([]uint64, header.Rank)
	if err := binary.Read(r, filmByteOrder, dimensions); err != nil {
		return filmFileMetadata{}, fmt.Errorf("shape: %w", err)
	}
	shape, err := decodeFilmShape(dimensions)
	if err != nil {
		return filmFileMetadata{}, err
	}

	var spectrum filmSpectrumHeader
	if err := binary.Read(r, filmByteOrder, &spectrum); err != nil {
		return filmFileMetadata{}, fmt.Errorf("spectrum header: %w", err)
	}
	metadata := filmFileMetadata{
		Samples:  header.Samples,
		Shape:    shape,
		BinCount: int(spectrum.BinCount),
		MinNM:    spectrum.MinNM,
		MaxNM:    spectrum.MaxNM,
	}
	if err := metadata.validate(); err != nil {
		return filmFileMetadata{}, err
	}
	return metadata, nil
}

func metadataForFilm(film *Film) (filmFileMetadata, error) {
	if film == nil {
		return filmFileMetadata{}, fmt.Errorf("cannot save a nil Film")
	}
	metadata := filmFileMetadata{
		Samples:  film.Samples,
		Shape:    film.Shape,
		BinCount: len(film.SpectralBins),
		MinNM:    film.SpectralMinNM,
		MaxNM:    film.SpectralMaxNM,
	}
	if err := metadata.validate(); err != nil {
		return filmFileMetadata{}, err
	}
	for bin := range film.SpectralBins {
		plane := &film.SpectralBins[bin]
		if !slices.Equal(plane.Shape, metadata.Shape) || len(plane.Data) != metadata.ElementCount {
			return filmFileMetadata{}, fmt.Errorf("spectral plane %d does not match Film shape %v", bin, metadata.Shape)
		}
	}
	return metadata, nil
}

func (metadata *filmFileMetadata) validate() error {
	if metadata.Samples < 0 {
		return fmt.Errorf("invalid sample count %d", metadata.Samples)
	}
	if len(metadata.Shape) == 0 || len(metadata.Shape) > int(maxFilmRank) {
		return fmt.Errorf("invalid rank %d", len(metadata.Shape))
	}
	elementCount, err := checkedElementCount(metadata.Shape)
	if err != nil {
		return err
	}
	if metadata.BinCount <= 0 || metadata.BinCount > int(maxFilmSpectralBins) {
		return fmt.Errorf("invalid spectral-bin count %d", metadata.BinCount)
	}
	if !validSpectralRange(metadata.MinNM, metadata.MaxNM) {
		return fmt.Errorf("invalid spectral range [%v, %v]", metadata.MinNM, metadata.MaxNM)
	}
	if _, err := checkedPayloadBytes(elementCount, metadata.BinCount); err != nil {
		return err
	}
	metadata.ElementCount = elementCount
	return nil
}

func decodeFilmShape(dimensions []uint64) ([]int, error) {
	maxInt := uint64(^uint(0) >> 1)
	shape := make([]int, len(dimensions))
	for axis, extent := range dimensions {
		if extent == 0 || extent > maxInt {
			return nil, fmt.Errorf("invalid dimension %d at axis %d", extent, axis)
		}
		shape[axis] = int(extent)
	}
	return shape, nil
}

func validSpectralRange(minNM, maxNM float64) bool {
	return minNM > 0 && maxNM > minNM &&
		!math.IsNaN(minNM) && !math.IsNaN(maxNM) &&
		!math.IsInf(minNM, 0) && !math.IsInf(maxNM, 0)
}

func checkedElementCount(shape []int) (int, error) {
	maxInt := int(^uint(0) >> 1)
	count := 1
	for axis, extent := range shape {
		if extent <= 0 || count > maxInt/extent {
			return 0, fmt.Errorf("invalid or overflowing Film dimension %d at axis %d", extent, axis)
		}
		count *= extent
	}
	return count, nil
}

func checkedPayloadBytes(elementCount, binCount int) (int64, error) {
	if elementCount <= 0 || binCount <= 0 {
		return 0, fmt.Errorf("invalid Film payload dimensions")
	}
	maxInt64 := uint64(^uint64(0) >> 1)
	if uint64(elementCount) > maxInt64/uint64(binCount)/8 {
		return 0, fmt.Errorf("Film payload is too large")
	}
	return int64(elementCount) * int64(binCount) * 8, nil
}
