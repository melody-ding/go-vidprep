package processor

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/melody-ding/go-vidprep/internal/types"
)

// createTestVideo creates a small test video file using ffmpeg
func createTestVideo(t *testing.T) string {
	// Create a temporary file for the test video
	tmpFile, err := os.CreateTemp("", "test-*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Create a 1-second test video with a solid color
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", "color=c=red:s=256x256:d=1",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		tmpFile.Name(),
		"-y",
	)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	return tmpFile.Name()
}

func TestParseDimensions(t *testing.T) {
	tests := []struct {
		name    string
		size    string
		want    Dimensions
		wantErr bool
	}{
		{
			name:    "valid dimensions",
			size:    "256x256",
			want:    Dimensions{Width: 256, Height: 256},
			wantErr: false,
		},
		{
			name:    "invalid format",
			size:    "256",
			want:    Dimensions{},
			wantErr: true,
		},
		{
			name:    "invalid width",
			size:    "abcx256",
			want:    Dimensions{},
			wantErr: true,
		},
		{
			name:    "invalid height",
			size:    "256xabc",
			want:    Dimensions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDimensions(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDimensions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && (got.Width != tt.want.Width || got.Height != tt.want.Height) {
				t.Errorf("parseDimensions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateNumpyHeader(t *testing.T) {
	tests := []struct {
		name  string
		shape []int
	}{
		{
			name:  "single dimension",
			shape: []int{10},
		},
		{
			name:  "multiple dimensions",
			shape: []int{10, 256, 256, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header, err := createNumpyHeader(tt.shape)
			if err != nil {
				t.Errorf("createNumpyHeader() error = %v", err)
				return
			}

			// Check magic string
			if string(header[0:6]) != "\x93NUMPY" {
				t.Error("createNumpyHeader() magic string incorrect")
			}

			// Check version
			if header[6] != 0x01 || header[7] != 0x00 {
				t.Error("createNumpyHeader() version incorrect")
			}

			// Check header length
			headerLen := binary.LittleEndian.Uint16(header[8:10])
			if headerLen == 0 {
				t.Error("createNumpyHeader() header length is 0")
			}
		})
	}
}

func TestProcessClip(t *testing.T) {
	// Create a test video file
	testVideoPath := createTestVideo(t)
	defer os.Remove(testVideoPath)

	// Read the test video data
	videoData, err := os.ReadFile(testVideoPath)
	if err != nil {
		t.Fatal(err)
	}

	// Create a temporary directory for test output
	tempDir, err := os.MkdirTemp("", "govidprep-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test clip with the actual video data
	clip := types.Clip{
		Key:     "test_clip",
		RawData: videoData,
	}

	// Test JPEG output
	t.Run("JPEG output", func(t *testing.T) {
		err := ProcessClip(clip, tempDir, 8, "256x256", FormatJPEG)
		if err != nil {
			t.Errorf("ProcessClip() error = %v", err)
		}

		// Check if output directory was created
		outputPath := filepath.Join(tempDir, clip.Key)
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("ProcessClip() did not create output directory")
		}

		// Check if at least one frame was created
		files, err := os.ReadDir(outputPath)
		if err != nil {
			t.Error("Failed to read output directory")
		}
		if len(files) == 0 {
			t.Error("No frames were created")
		}
	})

	// Test NPY output
	t.Run("NPY output", func(t *testing.T) {
		err := ProcessClip(clip, tempDir, 8, "256x256", FormatNPY)
		if err != nil {
			t.Errorf("ProcessClip() error = %v", err)
		}

		// Check if output file was created
		outputPath := filepath.Join(tempDir, clip.Key, "frames.npy")
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("ProcessClip() did not create NPY file")
		}

		// Check if the file has content
		fileInfo, err := os.Stat(outputPath)
		if err != nil {
			t.Error("Failed to stat NPY file")
		}
		if fileInfo.Size() == 0 {
			t.Error("NPY file is empty")
		}
	})
}
