package numpy

import (
	"os"
	"testing"
)

func TestWriter(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test-*.npy")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Create a writer
	writer, err := NewWriter(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()

	// Test data
	data := []byte{1, 2, 3, 4, 5, 6}
	shape := []int{2, 3}

	// Write the data
	if err := writer.Write(data, shape); err != nil {
		t.Fatal(err)
	}

	// Read the file back
	fileData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Check magic string
	if string(fileData[0:6]) != "\x93NUMPY" {
		t.Error("Invalid magic string in NPY file")
	}

	// Check version
	if fileData[6] != 0x01 || fileData[7] != 0x00 {
		t.Error("Invalid version in NPY file")
	}

	// Check that the data is present
	if len(fileData) <= len(data) {
		t.Error("File is too small to contain the data")
	}
}

func TestWriterWithDifferentShapes(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		shape []int
	}{
		{
			name:  "1D array",
			data:  []byte{1, 2, 3},
			shape: []int{3},
		},
		{
			name:  "2D array",
			data:  []byte{1, 2, 3, 4, 5, 6},
			shape: []int{2, 3},
		},
		{
			name:  "3D array",
			data:  []byte{1, 2, 3, 4, 5, 6, 7, 8},
			shape: []int{2, 2, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpFile, err := os.CreateTemp("", "test-*.npy")
			if err != nil {
				t.Fatal(err)
			}
			tmpFile.Close()
			defer os.Remove(tmpFile.Name())

			// Create a writer
			writer, err := NewWriter(tmpFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer writer.Close()

			// Write the data
			if err := writer.Write(tt.data, tt.shape); err != nil {
				t.Fatal(err)
			}

			// Check file size
			fileInfo, err := os.Stat(tmpFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			if fileInfo.Size() == 0 {
				t.Error("File is empty")
			}
		})
	}
}
