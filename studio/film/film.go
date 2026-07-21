package film

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	modelcamera "github.com/Algo2147483647/ray/engine/model/camera"
)

func MergeFilmFiles(basePath, updatePath, outputPath string) error {
	base, err := loadFilm(basePath)
	if err != nil {
		return fmt.Errorf("load resume film %q: %w", basePath, err)
	}
	update, err := loadFilm(updatePath)
	if err != nil {
		return fmt.Errorf("load rendered film %q: %w", updatePath, err)
	}

	if err := mergeFilms(base, update); err != nil {
		return err
	}
	if err := ensureParentDir(outputPath); err != nil {
		return err
	}
	if err := base.SaveToFile(outputPath); err != nil {
		return fmt.Errorf("save merged film %q: %w", outputPath, err)
	}
	return nil
}

func CopyFilmFile(sourcePath, outputPath string) error {
	film, err := loadFilm(sourcePath)
	if err != nil {
		return fmt.Errorf("load film %q: %w", sourcePath, err)
	}
	if err := ensureParentDir(outputPath); err != nil {
		return err
	}
	if err := film.SaveToFile(outputPath); err != nil {
		return fmt.Errorf("save film %q: %w", outputPath, err)
	}
	return nil
}

func SaveFilmImage(filmPath, imagePath string, options modelcamera.ImageOptions) error {
	film, err := loadFilm(filmPath)
	if err != nil {
		return fmt.Errorf("load film %q: %w", filmPath, err)
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
		return fmt.Errorf("film %q cannot be converted to an image", filmPath)
	}
	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("write image %q: %w", imagePath, err)
	}
	return nil
}

func loadFilm(path string) (*modelcamera.Film, error) {
	film := modelcamera.NewFilm()
	if err := film.LoadFromFile(path); err != nil {
		return nil, err
	}
	return film, nil
}

func mergeFilms(base, update *modelcamera.Film) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("merge films: %v", recovered)
		}
	}()
	base.Merge(update)
	return nil
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
