package binaryio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const float64Bytes = 8

// WriteFloat64s writes values with a reusable scratch buffer.
func WriteFloat64s(w io.Writer, order binary.ByteOrder, values []float64, scratch []byte) error {
	valuesPerChunk, err := float64ValuesPerChunk(scratch)
	if err != nil {
		return err
	}
	for start := 0; start < len(values); start += valuesPerChunk {
		end := min(start+valuesPerChunk, len(values))
		chunk := scratch[:(end-start)*float64Bytes]
		for i, value := range values[start:end] {
			order.PutUint64(chunk[i*float64Bytes:], math.Float64bits(value))
		}
		written, err := w.Write(chunk)
		if err != nil {
			return err
		}
		if written != len(chunk) {
			return io.ErrShortWrite
		}
	}
	return nil
}

// ReadFloat64s reads values with a reusable scratch buffer.
func ReadFloat64s(r io.Reader, order binary.ByteOrder, values []float64, scratch []byte) error {
	valuesPerChunk, err := float64ValuesPerChunk(scratch)
	if err != nil {
		return err
	}
	for start := 0; start < len(values); start += valuesPerChunk {
		end := min(start+valuesPerChunk, len(values))
		chunk := scratch[:(end-start)*float64Bytes]
		if _, err := io.ReadFull(r, chunk); err != nil {
			return err
		}
		for i := range values[start:end] {
			values[start+i] = math.Float64frombits(order.Uint64(chunk[i*float64Bytes:]))
		}
	}
	return nil
}

// RequireEOF rejects trailing bytes after a fully decoded payload.
func RequireEOF(r io.Reader) error {
	var extra [1]byte
	if _, err := io.ReadFull(r, extra[:]); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing data")
}

func float64ValuesPerChunk(scratch []byte) (int, error) {
	if len(scratch) < float64Bytes {
		return 0, fmt.Errorf("float64 scratch buffer must contain at least %d bytes", float64Bytes)
	}
	return len(scratch) / float64Bytes, nil
}
