package numpy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

// Writer handles writing data to NumPy (.npy) files
type Writer struct {
	file *os.File
}

// NewWriter creates a new NumPy writer for the given file
func NewWriter(filepath string) (*Writer, error) {
	file, err := os.Create(filepath)
	if err != nil {
		return nil, fmt.Errorf("error creating npy file: %v", err)
	}
	return &Writer{file: file}, nil
}

// Close closes the underlying file
func (w *Writer) Close() error {
	return w.file.Close()
}

// Write writes data to the NumPy file with the given shape
func (w *Writer) Write(data []byte, shape []int) error {
	// Create and write the header
	header, err := createHeader(shape)
	if err != nil {
		return fmt.Errorf("error creating numpy header: %v", err)
	}

	if _, err := w.file.Write(header); err != nil {
		return fmt.Errorf("error writing npy header: %v", err)
	}

	// Write the data
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("error writing npy data: %v", err)
	}

	return nil
}

// createHeader creates a NumPy array header with the given shape
func createHeader(shape []int) ([]byte, error) {
	// Create the dictionary string
	var shapeStr bytes.Buffer
	shapeStr.WriteString("{'descr': '<u1', 'fortran_order': False, 'shape': (")
	for i, s := range shape {
		shapeStr.WriteString(fmt.Sprintf("%d", s))
		if i < len(shape)-1 {
			shapeStr.WriteString(", ")
		}
	}
	shapeStr.WriteString(")}")

	dictBytes := shapeStr.Bytes()

	// Calculate padding for the dictionary string
	currentHeaderSize := len(dictBytes) + 10 // 10 = len(magic+version) + len(header_len_prefix)
	padding := (16 - (currentHeaderSize % 16)) % 16

	// Create the header
	var fullHeader bytes.Buffer

	// Magic string and version (NPY v1.0) - 8 bytes
	fullHeader.Write([]byte{0x93, 'N', 'U', 'M', 'P', 'Y', 0x01, 0x00})

	// Header length (uint16 little-endian) - 2 bytes
	headerDictWithPaddingLen := uint16(len(dictBytes) + padding)
	if err := binary.Write(&fullHeader, binary.LittleEndian, headerDictWithPaddingLen); err != nil {
		return nil, fmt.Errorf("failed to write header dictionary length: %v", err)
	}

	// Dictionary literal string
	fullHeader.Write(dictBytes)

	// Padding bytes
	fullHeader.Write(bytes.Repeat([]byte{' '}, padding))

	return fullHeader.Bytes(), nil
}
