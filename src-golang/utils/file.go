package utils

import (
	"bufio"
	"encoding/binary"
	"errors"
	"gonum.org/v1/gonum/mat"
	"io"
	"math"
	"os"
)

const magicString = "MAT3" // 文件标识符

// 保存三个矩阵到文件
func SaveMatrices(filename string, matrices [3]*mat.Dense) error {
	if len(matrices) != 3 {
		return errors.New("exactly 3 matrices required")
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// 写入文件头标识
	if _, err := writer.WriteString(magicString); err != nil {
		return err
	}

	// 写入矩阵元数据
	r, c := matrices[0].Dims()
	metadata := struct {
		Rows int64
		Cols int64
	}{int64(r), int64(c)}
	if err := binary.Write(writer, binary.LittleEndian, metadata); err != nil {
		return err
	}

	// 写入所有矩阵数据
	for _, m := range matrices {
		if err := writeMatrix(writer, m); err != nil {
			return err
		}
	}

	return nil
}

// 从文件加载三个矩阵
func LoadMatrices(filename string) ([3]*mat.Dense, error) {
	file, err := os.Open(filename)
	if err != nil {
		return [3]*mat.Dense{}, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// 验证文件格式
	magic := make([]byte, len(magicString))
	if _, err := io.ReadFull(reader, magic); err != nil {
		return [3]*mat.Dense{}, err
	}
	if string(magic) != magicString {
		return [3]*mat.Dense{}, errors.New("invalid file format")
	}

	// 读取矩阵元数据
	var metadata struct {
		Rows int64
		Cols int64
	}
	if err := binary.Read(reader, binary.LittleEndian, &metadata); err != nil {
		return [3]*mat.Dense{}, err
	}

	// 读取三个矩阵
	matrices := [3]*mat.Dense{}
	for i := range matrices {
		m, err := readMatrix(reader, int(metadata.Rows), int(metadata.Cols))
		if err != nil {
			return [3]*mat.Dense{}, err
		}
		matrices[i] = m
	}

	return matrices, nil
}

// 写入单个矩阵数据
func writeMatrix(w io.Writer, m *mat.Dense) error {
	r, c := m.Dims()
	data := m.RawMatrix().Data
	buf := make([]byte, r*c*8)

	for i, v := range data {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}

	_, err := w.Write(buf)
	return err
}

// 读取单个矩阵数据
func readMatrix(r io.Reader, rows, cols int) (*mat.Dense, error) {
	elements := rows * cols
	buf := make([]byte, elements*8)

	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}

	data := make([]float64, elements)
	for i := range data {
		data[i] = math.Float64frombits(binary.LittleEndian.Uint64(buf[i*8:]))
	}

	return mat.NewDense(rows, cols, data), nil
}
