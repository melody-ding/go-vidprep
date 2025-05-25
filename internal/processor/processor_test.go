package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/melody-ding/go-vidprep/internal/types"
)

// createTestVideo creates a small test video file using ffmpeg
func createTestVideo(t *testing.T, duration float64) string {
	// Create a temporary file for the test video
	tmpFile, err := os.CreateTemp("", "test-*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Create a test video with a solid color for specified duration
	cmd := exec.Command("ffmpeg",
		"-f", "lavfi",
		"-i", fmt.Sprintf("color=c=red:s=256x256:d=%.1f:r=8", duration), // Force input frame rate to 8fps
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-r", "8", // Force output frame rate to 8fps
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

func TestProcessClip(t *testing.T) {
	// Create a test video file (3 seconds at 8fps = 24 frames)
	testVideoPath := createTestVideo(t, 3.0)
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

	// Test JPEG output with chunking
	t.Run("JPEG output with chunking", func(t *testing.T) {
		targetFrames := 8 // Should create 3 chunks of 8 frames each
		err := ProcessClip(clip, tempDir, 8, "256x256", FormatJPEG, targetFrames)
		if err != nil {
			t.Errorf("ProcessClip() error = %v", err)
		}

		// Check if output directory was created
		outputPath := filepath.Join(tempDir, clip.Key)
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("ProcessClip() did not create output directory")
		}

		// Check chunk directories
		files, err := os.ReadDir(outputPath)
		if err != nil {
			t.Error("Failed to read output directory")
		}

		// Should have 3 chunk directories (24 frames total)
		if len(files) != 3 {
			t.Errorf("Expected 3 chunk directories, got %d", len(files))
		}

		// Check each chunk directory
		for i := 0; i < 3; i++ {
			chunkDir := filepath.Join(outputPath, fmt.Sprintf("chunk_%05d", i))
			if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
				t.Errorf("Chunk directory %s not found", chunkDir)
				continue
			}

			// Check frames in chunk
			frameFiles, err := os.ReadDir(chunkDir)
			if err != nil {
				t.Errorf("Failed to read chunk directory %s", chunkDir)
				continue
			}

			// Should have targetFrames + 1 (metadata file)
			if len(frameFiles) != targetFrames+1 {
				t.Errorf("Expected %d files in chunk %d, got %d", targetFrames+1, i, len(frameFiles))
			}

			// Check metadata file
			metadataPath := filepath.Join(chunkDir, "metadata.json")
			if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
				t.Errorf("Metadata file not found in chunk %d", i)
				continue
			}

			// Read and verify metadata
			metadataData, err := os.ReadFile(metadataPath)
			if err != nil {
				t.Errorf("Failed to read metadata file in chunk %d", i)
				continue
			}

			var metadata types.ClipMetadata
			if err := json.Unmarshal(metadataData, &metadata); err != nil {
				t.Errorf("Failed to parse metadata in chunk %d", i)
				continue
			}

			// Verify metadata fields
			if metadata.FrameCount != targetFrames {
				t.Errorf("Expected %d frames in metadata, got %d", targetFrames, metadata.FrameCount)
			}
			if metadata.FPS != 8 {
				t.Errorf("Expected FPS 8 in metadata, got %d", metadata.FPS)
			}
			if len(metadata.Size) != 2 || metadata.Size[0] != 256 || metadata.Size[1] != 256 {
				t.Errorf("Expected size [256, 256] in metadata, got %v", metadata.Size)
			}
		}
	})

	// Test NPY output with chunking
	t.Run("NPY output with chunking", func(t *testing.T) {
		targetFrames := 8 // Should create 3 chunks of 8 frames each
		err := ProcessClip(clip, tempDir, 8, "256x256", FormatNPY, targetFrames)
		if err != nil {
			t.Errorf("ProcessClip() error = %v", err)
		}

		// Check if output directory was created
		outputPath := filepath.Join(tempDir, clip.Key)
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("ProcessClip() did not create output directory")
		}

		// Check chunk files
		files, err := os.ReadDir(outputPath)
		if err != nil {
			t.Error("Failed to read output directory")
		}

		// Should have 6 files (3 .npy files and 3 metadata files)
		if len(files) != 6 {
			t.Errorf("Expected 6 files, got %d", len(files))
		}

		// Check each chunk
		for i := 0; i < 3; i++ {
			// Check NPY file
			npyPath := filepath.Join(outputPath, fmt.Sprintf("chunk_%05d.npy", i))
			if _, err := os.Stat(npyPath); os.IsNotExist(err) {
				t.Errorf("NPY file not found for chunk %d", i)
				continue
			}

			// Check metadata file
			metadataPath := filepath.Join(outputPath, fmt.Sprintf("chunk_%05d_metadata.json", i))
			if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
				t.Errorf("Metadata file not found for chunk %d", i)
				continue
			}

			// Read and verify metadata
			metadataData, err := os.ReadFile(metadataPath)
			if err != nil {
				t.Errorf("Failed to read metadata file for chunk %d", i)
				continue
			}

			var metadata types.ClipMetadata
			if err := json.Unmarshal(metadataData, &metadata); err != nil {
				t.Errorf("Failed to parse metadata for chunk %d", i)
				continue
			}

			// Verify metadata fields
			if metadata.FrameCount != targetFrames {
				t.Errorf("Expected %d frames in metadata, got %d", targetFrames, metadata.FrameCount)
			}
			if metadata.FPS != 8 {
				t.Errorf("Expected FPS 8 in metadata, got %d", metadata.FPS)
			}
			if len(metadata.Size) != 2 || metadata.Size[0] != 256 || metadata.Size[1] != 256 {
				t.Errorf("Expected size [256, 256] in metadata, got %v", metadata.Size)
			}
		}
	})
}

func TestProcessClipWithUnevenFrames(t *testing.T) {
	// Create a test video file (2.5 seconds at 8fps = 20 frames)
	testVideoPath := createTestVideo(t, 2.5)
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

	// Test with targetFrames that doesn't divide evenly
	targetFrames := 7 // Should create 2 chunks of 7 frames each, discard 6 frames

	t.Run("JPEG output with uneven frames", func(t *testing.T) {
		err := ProcessClip(clip, tempDir, 8, "256x256", FormatJPEG, targetFrames)
		if err != nil {
			t.Errorf("ProcessClip() error = %v", err)
		}

		// Check if output directory was created
		outputPath := filepath.Join(tempDir, clip.Key)
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("ProcessClip() did not create output directory")
		}

		// Check chunk directories
		files, err := os.ReadDir(outputPath)
		if err != nil {
			t.Error("Failed to read output directory")
		}

		// Should have 2 chunk directories (14 frames total, 6 frames discarded)
		if len(files) != 2 {
			t.Errorf("Expected 2 chunk directories, got %d", len(files))
		}

		// Check each chunk directory
		for i := 0; i < 2; i++ {
			chunkDir := filepath.Join(outputPath, fmt.Sprintf("chunk_%05d", i))
			if _, err := os.Stat(chunkDir); os.IsNotExist(err) {
				t.Errorf("Chunk directory %s not found", chunkDir)
				continue
			}

			// Check frames in chunk
			frameFiles, err := os.ReadDir(chunkDir)
			if err != nil {
				t.Errorf("Failed to read chunk directory %s", chunkDir)
				continue
			}

			// Should have targetFrames + 1 (metadata file)
			if len(frameFiles) != targetFrames+1 {
				t.Errorf("Expected %d files in chunk %d, got %d", targetFrames+1, i, len(frameFiles))
			}
		}
	})

	t.Run("NPY output with uneven frames", func(t *testing.T) {
		err := ProcessClip(clip, tempDir, 8, "256x256", FormatNPY, targetFrames)
		if err != nil {
			t.Errorf("ProcessClip() error = %v", err)
		}

		// Check if output directory was created
		outputPath := filepath.Join(tempDir, clip.Key)
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Error("ProcessClip() did not create output directory")
		}

		// Check chunk files
		files, err := os.ReadDir(outputPath)
		if err != nil {
			t.Error("Failed to read output directory")
		}

		// Should have 4 files (2 .npy files and 2 metadata files)
		if len(files) != 4 {
			t.Errorf("Expected 4 files, got %d", len(files))
		}

		// Check each chunk
		for i := 0; i < 2; i++ {
			// Check NPY file
			npyPath := filepath.Join(outputPath, fmt.Sprintf("chunk_%05d.npy", i))
			if _, err := os.Stat(npyPath); os.IsNotExist(err) {
				t.Errorf("NPY file not found for chunk %d", i)
				continue
			}

			// Check metadata file
			metadataPath := filepath.Join(outputPath, fmt.Sprintf("chunk_%05d_metadata.json", i))
			if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
				t.Errorf("Metadata file not found for chunk %d", i)
				continue
			}
		}
	})
}
