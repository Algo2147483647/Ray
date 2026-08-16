package camera

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"slices"

	"github.com/Algo2147483647/ray/engine/maths"
)

var filmFileMagic = [8]byte{'R', 'A', 'Y', 'F', 'I', 'L', 'M', 0}

const (
	filmFileVersion     uint32 = 3
	filmFloatChunkBytes        = 1 << 20
	maxFilmRank         uint32 = 16
	maxFilmSpectralBins uint32 = 4096
)

func (f *Film) SaveToFile(filename string) error {
	shape, elementCount, spectralCount, err := validateFilmForWrite(f)
	if err != nil {
		return err
	}
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	writeErr := f.writeFilm(file, shape, elementCount, spectralCount)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func (f *Film) writeFilm(w io.Writer, shape []int, elementCount int, spectralCount uint32) error {
	if err := writeFull(w, filmFileMagic[:]); err != nil {
		return err
	}
	for _, value := range []interface{}{filmFileVersion, f.Samples, uint32(len(shape))} {
		if err := binary.Write(w, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	for _, extent := range shape {
		if err := binary.Write(w, binary.LittleEndian, uint64(extent)); err != nil {
			return err
		}
	}
	for _, value := range []interface{}{spectralCount, f.SpectralMinNM, f.SpectralMaxNM} {
		if err := binary.Write(w, binary.LittleEndian, value); err != nil {
			return err
		}
	}
	buffer := make([]byte, filmFloatChunkBytes)
	for bin := range f.SpectralBins {
		if err := writeFloat64Blocks(w, f.SpectralBins[bin].Data[:elementCount], buffer); err != nil {
			return fmt.Errorf("write spectral plane %d: %w", bin, err)
		}
	}
	return nil
}

func (f *Film) LoadFromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	return f.readFilm(file, stat.Size())
}

func (f *Film) readFilm(file *os.File, fileSize int64) error {
	var magic [8]byte
	if _, err := io.ReadFull(file, magic[:]); err != nil {
		return fmt.Errorf("read Film magic: %w", err)
	}
	if magic != filmFileMagic {
		return fmt.Errorf("unsupported Film file: invalid magic")
	}
	var version uint32
	if err := binary.Read(file, binary.LittleEndian, &version); err != nil {
		return fmt.Errorf("read Film version: %w", err)
	}
	if version != filmFileVersion {
		return fmt.Errorf("unsupported Film version %d; expected %d", version, filmFileVersion)
	}
	var samples int64
	if err := binary.Read(file, binary.LittleEndian, &samples); err != nil {
		return fmt.Errorf("read Film sample count: %w", err)
	}
	if samples < 0 {
		return fmt.Errorf("invalid Film sample count %d", samples)
	}
	var rank uint32
	if err := binary.Read(file, binary.LittleEndian, &rank); err != nil {
		return fmt.Errorf("read Film rank: %w", err)
	}
	if rank == 0 || rank > maxFilmRank {
		return fmt.Errorf("invalid Film rank %d", rank)
	}
	shape := make([]int, rank)
	maxInt := uint64(^uint(0) >> 1)
	for i := range shape {
		var extent uint64
		if err := binary.Read(file, binary.LittleEndian, &extent); err != nil {
			return fmt.Errorf("read Film dimension %d: %w", i, err)
		}
		if extent == 0 || extent > maxInt {
			return fmt.Errorf("invalid Film dimension %d at axis %d", extent, i)
		}
		shape[i] = int(extent)
	}
	elementCount, err := checkedElementCount(shape)
	if err != nil {
		return err
	}
	var spectralCount uint32
	if err := binary.Read(file, binary.LittleEndian, &spectralCount); err != nil {
		return fmt.Errorf("read Film spectral-bin count: %w", err)
	}
	if spectralCount == 0 || spectralCount > maxFilmSpectralBins {
		return fmt.Errorf("invalid Film spectral-bin count %d", spectralCount)
	}
	var spectralMin, spectralMax float64
	if err := binary.Read(file, binary.LittleEndian, &spectralMin); err != nil {
		return fmt.Errorf("read Film spectral minimum: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &spectralMax); err != nil {
		return fmt.Errorf("read Film spectral maximum: %w", err)
	}
	if !(spectralMin > 0) || !(spectralMax > spectralMin) || math.IsInf(spectralMin, 0) || math.IsInf(spectralMax, 0) {
		return fmt.Errorf("invalid Film spectral range [%v, %v]", spectralMin, spectralMax)
	}
	position, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	expectedBytes, err := checkedPlaneBytes(elementCount, int(spectralCount))
	if err != nil {
		return err
	}
	if remaining := fileSize - position; remaining != expectedBytes {
		return fmt.Errorf("invalid Film payload size %d; expected %d", remaining, expectedBytes)
	}
	bins := make([]maths.Tensor[float64], spectralCount)
	for i := range bins {
		bins[i] = *maths.NewTensor[float64](shape)
	}
	buffer := make([]byte, filmFloatChunkBytes)
	for bin := range bins {
		if err := readFloat64Blocks(file, bins[bin].Data, buffer); err != nil {
			return fmt.Errorf("read spectral plane %d: %w", bin, err)
		}
	}
	f.Shape = shape
	f.Samples = samples
	f.SpectralBins = bins
	f.SpectralMinNM = spectralMin
	f.SpectralMaxNM = spectralMax
	return nil
}

func validateFilmForWrite(f *Film) ([]int, int, uint32, error) {
	if f == nil {
		return nil, 0, 0, fmt.Errorf("cannot save a nil Film")
	}
	if len(f.Shape) == 0 || len(f.Shape) > int(maxFilmRank) {
		return nil, 0, 0, fmt.Errorf("invalid Film rank %d", len(f.Shape))
	}
	elementCount, err := checkedElementCount(f.Shape)
	if err != nil {
		return nil, 0, 0, err
	}
	if f.Samples < 0 {
		return nil, 0, 0, fmt.Errorf("invalid Film sample count %d", f.Samples)
	}
	if len(f.SpectralBins) == 0 || len(f.SpectralBins) > int(maxFilmSpectralBins) {
		return nil, 0, 0, fmt.Errorf("invalid Film spectral-bin count %d", len(f.SpectralBins))
	}
	if !(f.SpectralMinNM > 0) || !(f.SpectralMaxNM > f.SpectralMinNM) ||
		math.IsInf(f.SpectralMinNM, 0) || math.IsInf(f.SpectralMaxNM, 0) {
		return nil, 0, 0, fmt.Errorf("invalid Film spectral range [%v, %v]", f.SpectralMinNM, f.SpectralMaxNM)
	}
	for bin := range f.SpectralBins {
		if !slices.Equal(f.SpectralBins[bin].Shape, f.Shape) || len(f.SpectralBins[bin].Data) != elementCount {
			return nil, 0, 0, fmt.Errorf("Film spectral plane %d does not match shape %v", bin, f.Shape)
		}
	}
	if _, err := checkedPlaneBytes(elementCount, len(f.SpectralBins)); err != nil {
		return nil, 0, 0, err
	}
	return f.Shape, elementCount, uint32(len(f.SpectralBins)), nil
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

func checkedPlaneBytes(elementCount, planeCount int) (int64, error) {
	if elementCount <= 0 || planeCount <= 0 {
		return 0, fmt.Errorf("invalid Film payload dimensions")
	}
	maxInt64 := uint64(^uint64(0) >> 1)
	if uint64(elementCount) > maxInt64/uint64(planeCount)/8 {
		return 0, fmt.Errorf("Film payload is too large")
	}
	return int64(uint64(elementCount) * uint64(planeCount) * 8), nil
}

func writeFloat64Blocks(w io.Writer, values []float64, buffer []byte) error {
	valuesPerBlock := len(buffer) / 8
	for start := 0; start < len(values); start += valuesPerBlock {
		end := min(start+valuesPerBlock, len(values))
		block := buffer[:(end-start)*8]
		for i, value := range values[start:end] {
			binary.LittleEndian.PutUint64(block[i*8:], math.Float64bits(value))
		}
		if err := writeFull(w, block); err != nil {
			return err
		}
	}
	return nil
}

func readFloat64Blocks(r io.Reader, values []float64, buffer []byte) error {
	valuesPerBlock := len(buffer) / 8
	for start := 0; start < len(values); start += valuesPerBlock {
		end := min(start+valuesPerBlock, len(values))
		block := buffer[:(end-start)*8]
		if _, err := io.ReadFull(r, block); err != nil {
			return err
		}
		for i := range values[start:end] {
			values[start+i] = math.Float64frombits(binary.LittleEndian.Uint64(block[i*8:]))
		}
	}
	return nil
}

func writeFull(w io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := w.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}
