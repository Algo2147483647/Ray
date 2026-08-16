package binaryio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"slices"
	"testing"
)

func TestFloat64sRoundTripAcrossChunks(t *testing.T) {
	want := []float64{1.25, -2.5, 3.75, 4.125, -5.5}
	var encoded bytes.Buffer
	if err := WriteFloat64s(&encoded, binary.LittleEndian, want, make([]byte, 16)); err != nil {
		t.Fatal(err)
	}
	got := make([]float64, len(want))
	if err := ReadFloat64s(&encoded, binary.LittleEndian, got, make([]byte, 24)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWriteFloat64sRejectsShortWrite(t *testing.T) {
	err := WriteFloat64s(shortWriter{}, binary.LittleEndian, []float64{1}, make([]byte, 8))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("got %v, want io.ErrShortWrite", err)
	}
}

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) {
	return len(data) - 1, nil
}
