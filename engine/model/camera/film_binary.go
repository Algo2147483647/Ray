package camera

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"

	"github.com/Algo2147483647/ray/engine/maths"
)

var filmFileMagic = [8]byte{'R', 'A', 'Y', 'F', 'I', 'L', 'M', 0}

const (
	filmFileVersion      uint32 = 2
	filmFloatChunkBytes         = 1 << 20
	maxFilmRank          uint32 = 16
	maxFilmColorSpaceLen uint32 = 64
	maxFilmSpectralBins  uint32 = 4096
)

// SaveToFile writes the current, intentionally versioned Film format. Float
// planes are encoded in reusable 1 MiB blocks instead of one binary.Write call
// per value.
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
	if err := binary.Write(w, binary.LittleEndian, filmFileVersion); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, f.Samples); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(shape))); err != nil {
		return err
	}
	for _, dim := range shape {
		if err := binary.Write(w, binary.LittleEndian, uint64(dim)); err != nil {
			return err
		}
	}

	space := []byte(f.ColorSpace)
	if err := binary.Write(w, binary.LittleEndian, uint32(len(space))); err != nil {
		return err
	}
	if err := writeFull(w, space); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, spectralCount); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, f.SpectralMinNM); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, f.SpectralMaxNM); err != nil {
		return err
	}

	buffer := make([]byte, filmFloatChunkBytes)
	for channel := range f.Data {
		if err := writeFloat64Blocks(w, f.Data[channel].Data[:elementCount], buffer); err != nil {
			return fmt.Errorf("write color plane %d: %w", channel, err)
		}
	}
	for bin := range f.SpectralBins {
		if err := writeFloat64Blocks(w, f.SpectralBins[bin].Data[:elementCount], buffer); err != nil {
			return fmt.Errorf("write spectral plane %d: %w", bin, err)
		}
	}
	return nil
}

// LoadFromFile accepts only the current Film format. Legacy headerless streams
// are deliberately rejected so a corrupt or obsolete file cannot be guessed.
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
		return fmt.Errorf("read Film samples: %w", err)
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
		var dim uint64
		if err := binary.Read(file, binary.LittleEndian, &dim); err != nil {
			return fmt.Errorf("read Film dimension %d: %w", i, err)
		}
		if dim == 0 || dim > maxInt {
			return fmt.Errorf("invalid Film dimension %d at axis %d", dim, i)
		}
		shape[i] = int(dim)
	}
	elementCount, err := checkedElementCount(shape)
	if err != nil {
		return err
	}

	var colorSpaceLen uint32
	if err := binary.Read(file, binary.LittleEndian, &colorSpaceLen); err != nil {
		return fmt.Errorf("read Film color-space length: %w", err)
	}
	if colorSpaceLen == 0 || colorSpaceLen > maxFilmColorSpaceLen {
		return fmt.Errorf("invalid Film color-space length %d", colorSpaceLen)
	}
	colorSpaceBytes := make([]byte, colorSpaceLen)
	if _, err := io.ReadFull(file, colorSpaceBytes); err != nil {
		return fmt.Errorf("read Film color space: %w", err)
	}
	colorSpace := FilmColorSpace(colorSpaceBytes)
	if !validFilmColorSpace(colorSpace) {
		return fmt.Errorf("unsupported Film color space %q", colorSpace)
	}

	var spectralCount uint32
	if err := binary.Read(file, binary.LittleEndian, &spectralCount); err != nil {
		return fmt.Errorf("read Film spectral-bin count: %w", err)
	}
	if spectralCount > maxFilmSpectralBins {
		return fmt.Errorf("invalid Film spectral-bin count %d", spectralCount)
	}
	var spectralMin, spectralMax float64
	if err := binary.Read(file, binary.LittleEndian, &spectralMin); err != nil {
		return fmt.Errorf("read Film spectral minimum: %w", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &spectralMax); err != nil {
		return fmt.Errorf("read Film spectral maximum: %w", err)
	}
	if spectralCount > 0 && (!(spectralMin > 0) || !(spectralMax > spectralMin) || math.IsInf(spectralMin, 0) || math.IsInf(spectralMax, 0)) {
		return fmt.Errorf("invalid Film spectral range [%v, %v]", spectralMin, spectralMax)
	}

	position, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	expectedBytes, err := checkedPlaneBytes(elementCount, int(spectralCount)+3)
	if err != nil {
		return err
	}
	if remaining := fileSize - position; remaining != expectedBytes {
		return fmt.Errorf("invalid Film payload size %d; expected %d", remaining, expectedBytes)
	}

	data := [3]maths.Tensor[float64]{
		*maths.NewTensor[float64](shape),
		*maths.NewTensor[float64](shape),
		*maths.NewTensor[float64](shape),
	}
	spectralBins := make([]maths.Tensor[float64], spectralCount)
	for i := range spectralBins {
		spectralBins[i] = *maths.NewTensor[float64](shape)
	}

	buffer := make([]byte, filmFloatChunkBytes)
	for channel := range data {
		if err := readFloat64Blocks(file, data[channel].Data, buffer); err != nil {
			return fmt.Errorf("read color plane %d: %w", channel, err)
		}
	}
	for bin := range spectralBins {
		if err := readFloat64Blocks(file, spectralBins[bin].Data, buffer); err != nil {
			return fmt.Errorf("read spectral plane %d: %w", bin, err)
		}
	}

	f.Data = data
	f.Samples = samples
	f.ColorSpace = colorSpace
	f.SpectralBins = spectralBins
	if spectralCount > 0 {
		f.SpectralMinNM = spectralMin
		f.SpectralMaxNM = spectralMax
	} else {
		f.SpectralMinNM = 0
		f.SpectralMaxNM = 0
	}
	return nil
}

func validateFilmForWrite(f *Film) ([]int, int, uint32, error) {
	if f == nil {
		return nil, 0, 0, fmt.Errorf("cannot save a nil Film")
	}
	shape := f.Data[0].Shape
	if len(shape) == 0 || len(shape) > int(maxFilmRank) {
		return nil, 0, 0, fmt.Errorf("invalid Film rank %d", len(shape))
	}
	elementCount, err := checkedElementCount(shape)
	if err != nil {
		return nil, 0, 0, err
	}
	if f.Samples < 0 {
		return nil, 0, 0, fmt.Errorf("invalid Film sample count %d", f.Samples)
	}
	if !validFilmColorSpace(f.ColorSpace) {
		return nil, 0, 0, fmt.Errorf("unsupported Film color space %q", f.ColorSpace)
	}
	for channel := range f.Data {
		if !reflect.DeepEqual(f.Data[channel].Shape, shape) || len(f.Data[channel].Data) != elementCount {
			return nil, 0, 0, fmt.Errorf("Film color plane %d does not match shape %v", channel, shape)
		}
	}
	if len(f.SpectralBins) > int(maxFilmSpectralBins) {
		return nil, 0, 0, fmt.Errorf("invalid Film spectral-bin count %d", len(f.SpectralBins))
	}
	if len(f.SpectralBins) > 0 && (!(f.SpectralMinNM > 0) || !(f.SpectralMaxNM > f.SpectralMinNM) || math.IsInf(f.SpectralMinNM, 0) || math.IsInf(f.SpectralMaxNM, 0)) {
		return nil, 0, 0, fmt.Errorf("invalid Film spectral range [%v, %v]", f.SpectralMinNM, f.SpectralMaxNM)
	}
	for bin := range f.SpectralBins {
		if !reflect.DeepEqual(f.SpectralBins[bin].Shape, shape) || len(f.SpectralBins[bin].Data) != elementCount {
			return nil, 0, 0, fmt.Errorf("Film spectral plane %d does not match shape %v", bin, shape)
		}
	}
	if _, err := checkedPlaneBytes(elementCount, len(f.SpectralBins)+3); err != nil {
		return nil, 0, 0, err
	}
	return shape, elementCount, uint32(len(f.SpectralBins)), nil
}

func checkedElementCount(shape []int) (int, error) {
	maxInt := int(^uint(0) >> 1)
	count := 1
	for axis, dim := range shape {
		if dim <= 0 || count > maxInt/dim {
			return 0, fmt.Errorf("invalid or overflowing Film dimension %d at axis %d", dim, axis)
		}
		count *= dim
	}
	return count, nil
}

func checkedPlaneBytes(elementCount, planeCount int) (int64, error) {
	if elementCount <= 0 || planeCount < 3 {
		return 0, fmt.Errorf("invalid Film payload dimensions")
	}
	maxInt64 := uint64(^uint64(0) >> 1)
	if uint64(elementCount) > maxInt64/uint64(planeCount)/8 {
		return 0, fmt.Errorf("Film payload is too large")
	}
	bytes := uint64(elementCount) * uint64(planeCount) * 8
	return int64(bytes), nil
}

func validFilmColorSpace(space FilmColorSpace) bool {
	switch space {
	case FilmColorSpaceLinearSRGB, FilmColorSpaceXYZ, FilmColorSpaceACEScg:
		return true
	default:
		return false
	}
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
