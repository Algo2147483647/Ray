package film

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
)

var filmFileMagic = [8]byte{'R', 'A', 'Y', 'F', 'I', 'L', 'M', 0}

const filmFileVersion uint32 = 3
const maxFilmRank uint32 = 16
const filmFloatChunkBytes = 1 << 20

var filmByteOrder = binary.LittleEndian

type filmFileHeader struct {
	Magic   [8]byte
	Version uint32
	Samples int64
	Rank    uint32
}
type filmSpectrumHeader struct {
	BinCount     uint32
	MinNM, MaxNM float64
}

func (film *Film) SaveToFile(filename string) error {
	if err := validateFilm(film); err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create Film %q: %w", filename, err)
	}
	if err := writeFilm(file, film); err != nil {
		_ = file.Close()
		return fmt.Errorf("write Film %q: %w", filename, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close Film %q: %w", filename, err)
	}
	return nil
}

func (film *Film) LoadFromFile(filename string) error {
	if film == nil {
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
	*film = *loaded
	return nil
}

func writeFilm(writer io.Writer, film *Film) error {
	header := filmFileHeader{Magic: filmFileMagic, Version: filmFileVersion, Samples: film.Samples, Rank: uint32(len(film.Shape))}
	if err := binary.Write(writer, filmByteOrder, header); err != nil {
		return fmt.Errorf("header: %w", err)
	}
	dimensions := make([]uint64, len(film.Shape))
	for index, extent := range film.Shape {
		dimensions[index] = uint64(extent)
	}
	if err := binary.Write(writer, filmByteOrder, dimensions); err != nil {
		return fmt.Errorf("shape: %w", err)
	}
	spectrum := filmSpectrumHeader{BinCount: uint32(len(film.SpectralBins)), MinNM: film.SpectralMinNM, MaxNM: film.SpectralMaxNM}
	if err := binary.Write(writer, filmByteOrder, spectrum); err != nil {
		return fmt.Errorf("spectrum header: %w", err)
	}
	scratch := make([]byte, filmFloatChunkBytes)
	for bin := range film.SpectralBins {
		if err := writeFloat64s(writer, film.SpectralBins[bin].Data, scratch); err != nil {
			return fmt.Errorf("spectral plane %d: %w", bin, err)
		}
	}
	return nil
}

func readFilm(reader io.Reader) (*Film, error) {
	var header filmFileHeader
	if err := binary.Read(reader, filmByteOrder, &header); err != nil {
		return nil, fmt.Errorf("header: %w", err)
	}
	if header.Magic != filmFileMagic {
		return nil, fmt.Errorf("invalid magic")
	}
	if header.Version != filmFileVersion {
		return nil, fmt.Errorf("unsupported version %d; expected %d", header.Version, filmFileVersion)
	}
	if header.Rank == 0 || header.Rank > maxFilmRank {
		return nil, fmt.Errorf("invalid rank %d", header.Rank)
	}
	dimensions := make([]uint64, header.Rank)
	if err := binary.Read(reader, filmByteOrder, dimensions); err != nil {
		return nil, fmt.Errorf("shape: %w", err)
	}
	shape := make([]int, len(dimensions))
	maximumInt := uint64(^uint(0) >> 1)
	for index, extent := range dimensions {
		if extent == 0 || extent > maximumInt {
			return nil, fmt.Errorf("invalid dimension %d at axis %d", extent, index)
		}
		shape[index] = int(extent)
	}
	var spectrum filmSpectrumHeader
	if err := binary.Read(reader, filmByteOrder, &spectrum); err != nil {
		return nil, fmt.Errorf("spectrum header: %w", err)
	}
	film := NewFilm(shape...)
	film.Samples = header.Samples
	film.InitSpectralBins(int(spectrum.BinCount), spectrum.MinNM, spectrum.MaxNM)
	if err := validateFilm(film); err != nil {
		return nil, err
	}
	scratch := make([]byte, filmFloatChunkBytes)
	for bin := range film.SpectralBins {
		if err := readFloat64s(reader, film.SpectralBins[bin].Data, scratch); err != nil {
			return nil, fmt.Errorf("spectral plane %d: %w", bin, err)
		}
	}
	var extra [1]byte
	if _, err := io.ReadFull(reader, extra[:]); err != io.EOF {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected trailing data")
	}
	return film, nil
}

func validateFilm(film *Film) error {
	if film == nil {
		return fmt.Errorf("cannot save a nil Film")
	}
	if film.Samples < 0 {
		return fmt.Errorf("invalid sample count %d", film.Samples)
	}
	if len(film.Shape) == 0 || len(film.Shape) > int(maxFilmRank) {
		return fmt.Errorf("invalid rank %d", len(film.Shape))
	}
	elements, err := checkedElementCount(film.Shape)
	if err != nil {
		return err
	}
	if len(film.SpectralBins) == 0 || len(film.SpectralBins) > MaxSpectralBinCount {
		return fmt.Errorf("invalid spectral-bin count %d", len(film.SpectralBins))
	}
	if !validSpectralRange(film.SpectralMinNM, film.SpectralMaxNM) {
		return fmt.Errorf("invalid spectral range [%v, %v]", film.SpectralMinNM, film.SpectralMaxNM)
	}
	for bin, plane := range film.SpectralBins {
		if !slices.Equal(plane.Shape, film.Shape) || len(plane.Data) != elements {
			return fmt.Errorf("spectral plane %d does not match Film shape %v", bin, film.Shape)
		}
	}
	return nil
}

func checkedElementCount(shape []int) (int, error) {
	maximum := int(^uint(0) >> 1)
	count := 1
	for axis, extent := range shape {
		if extent <= 0 || count > maximum/extent {
			return 0, fmt.Errorf("invalid or overflowing Film dimension %d at axis %d", extent, axis)
		}
		count *= extent
	}
	return count, nil
}
func writeFloat64s(writer io.Writer, values []float64, scratch []byte) error {
	perChunk := len(scratch) / 8
	for start := 0; start < len(values); start += perChunk {
		end := min(start+perChunk, len(values))
		chunk := scratch[:(end-start)*8]
		for index, value := range values[start:end] {
			filmByteOrder.PutUint64(chunk[index*8:], math.Float64bits(value))
		}
		written, err := writer.Write(chunk)
		if err != nil {
			return err
		}
		if written != len(chunk) {
			return io.ErrShortWrite
		}
	}
	return nil
}
func readFloat64s(reader io.Reader, values []float64, scratch []byte) error {
	perChunk := len(scratch) / 8
	for start := 0; start < len(values); start += perChunk {
		end := min(start+perChunk, len(values))
		chunk := scratch[:(end-start)*8]
		if _, err := io.ReadFull(reader, chunk); err != nil {
			return err
		}
		for index := range values[start:end] {
			values[start+index] = math.Float64frombits(filmByteOrder.Uint64(chunk[index*8:]))
		}
	}
	return nil
}
