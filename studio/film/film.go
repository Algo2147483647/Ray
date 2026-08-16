package film

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

func MergeFilmFiles(basePath, updatePath, outputPath string) error {
	base, err := LoadFilm(basePath)
	if err != nil {
		return fmt.Errorf("load resume film %q: %w", basePath, err)
	}
	update, err := LoadFilm(updatePath)
	if err != nil {
		return fmt.Errorf("load rendered film %q: %w", updatePath, err)
	}

	if err := MergeFilms(base, update); err != nil {
		return err
	}
	return SaveFilm(base, outputPath)
}

func MergeFilmFilesWithPixelWindows(basePath, updatePath, outputPath string, windows []modelcamera.PixelWindow) error {
	if len(windows) == 0 {
		return MergeFilmFiles(basePath, updatePath, outputPath)
	}

	base, err := LoadFilm(basePath)
	if err != nil {
		return fmt.Errorf("load resume film %q: %w", basePath, err)
	}
	update, err := LoadFilm(updatePath)
	if err != nil {
		return fmt.Errorf("load rendered film %q: %w", updatePath, err)
	}

	if err := MergeFilmsWithPixelWindows(base, update, windows); err != nil {
		return err
	}
	return SaveFilm(base, outputPath)
}

func CopyFilmFile(sourcePath, outputPath string) error {
	film, err := LoadFilm(sourcePath)
	if err != nil {
		return fmt.Errorf("load film %q: %w", sourcePath, err)
	}
	return SaveFilm(film, outputPath)
}

func SaveFilmImage(filmPath, imagePath string, options modelcamera.ImageOptions) error {
	film, err := LoadFilm(filmPath)
	if err != nil {
		return fmt.Errorf("load film %q: %w", filmPath, err)
	}
	return SaveFilmImageFromFilm(film, imagePath, options)
}

func SaveFilmImageFromFilm(film *modelcamera.Film, imagePath string, options modelcamera.ImageOptions) error {
	if film == nil {
		return fmt.Errorf("cannot create image from a nil film")
	}
	if err := ensureParentDir(imagePath); err != nil {
		return err
	}

	file, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("create image %q: %w", imagePath, err)
	}
	defer file.Close()

	img := film.ToImageWithOptions(options)
	if img == nil {
		return fmt.Errorf("film cannot be converted to an image")
	}
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("write image %q: %w", imagePath, err)
	}
	return nil
}

func LoadFilm(path string) (*modelcamera.Film, error) {
	film := modelcamera.NewFilm()
	if err := film.LoadFromFile(path); err != nil {
		return nil, err
	}
	return film, nil
}

func SaveFilm(film *modelcamera.Film, path string) error {
	if film == nil {
		return fmt.Errorf("cannot save a nil film")
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}
	if err := film.SaveToFile(path); err != nil {
		return fmt.Errorf("save film %q: %w", path, err)
	}
	return nil
}

func MergeFilms(base, update *modelcamera.Film) (err error) {
	if base == nil || update == nil {
		return fmt.Errorf("merge films: nil film")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("merge films: %v", recovered)
		}
	}()
	base.Merge(update)
	return nil
}

func MergeFilmsWithPixelWindows(base, update *modelcamera.Film, windows []modelcamera.PixelWindow) error {
	if base == nil || update == nil {
		return fmt.Errorf("merge films: nil film")
	}
	if len(windows) == 0 {
		return MergeFilms(base, update)
	}
	normalized, err := modelcamera.NormalizePixelWindows(windows, base.Data[0].Shape)
	if err != nil {
		return err
	}
	return mergeFilmsAtPixelWindows(base, update, normalized)
}

func mergeFilmsAtPixelWindows(base, update *modelcamera.Film, windows []modelcamera.PixelWindow) error {
	if !reflect.DeepEqual(base.Data[0].Shape, update.Data[0].Shape) {
		return fmt.Errorf("merge films: dimension of a and b is not matched")
	}
	if base.ColorSpace != "" && update.ColorSpace != "" && base.ColorSpace != update.ColorSpace {
		return fmt.Errorf("merge films: working space of a and b is not matched")
	}

	totalSamples := base.Samples + update.Samples
	if totalSamples == 0 {
		return nil
	}

	for _, pixel := range pixelWindowIndices(base.Data[0].Shape, windows) {
		for ch := 0; ch < 3; ch++ {
			base.Data[ch].Data[pixel] = (base.Data[ch].Data[pixel]*float64(base.Samples) + update.Data[ch].Data[pixel]*float64(update.Samples)) / float64(totalSamples)
		}
		if compatibleSpectralBins(base, update) {
			for bin := range base.SpectralBins {
				base.SpectralBins[bin].Data[pixel] = (base.SpectralBins[bin].Data[pixel]*float64(base.Samples) + update.SpectralBins[bin].Data[pixel]*float64(update.Samples)) / float64(totalSamples)
			}
		}
	}
	base.Samples = totalSamples
	return nil
}

func compatibleSpectralBins(base, update *modelcamera.Film) bool {
	return len(base.SpectralBins) > 0 &&
		len(base.SpectralBins) == len(update.SpectralBins) &&
		base.SpectralMinNM == update.SpectralMinNM &&
		base.SpectralMaxNM == update.SpectralMaxNM
}

func pixelWindowIndices(shape []int, windows []modelcamera.PixelWindow) []int {
	total := 1
	strides := make([]int, len(shape))
	for i, dim := range shape {
		strides[i] = total
		total *= dim
	}

	seen := make([]bool, total)
	indices := make([]int, 0)
	var walk func(window modelcamera.PixelWindow, dim, index int)
	walk = func(window modelcamera.PixelWindow, dim, index int) {
		if dim == len(shape) {
			if !seen[index] {
				seen[index] = true
				indices = append(indices, index)
			}
			return
		}
		for coord := window.Min[dim]; coord < window.Max[dim]; coord++ {
			walk(window, dim+1, index+coord*strides[dim])
		}
	}
	for _, window := range windows {
		walk(window, 0, 0)
	}
	return indices
}

func ensureParentDir(filename string) error {
	dir := filepath.Dir(filename)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", filename, err)
	}
	return nil
}
