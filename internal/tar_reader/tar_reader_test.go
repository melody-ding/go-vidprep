package tar_reader

import (
	"archive/tar"
	"bytes"
	"os"
	"testing"
)

func createTestTar(t *testing.T) *bytes.Buffer {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	// Add a video file
	header := &tar.Header{
		Name: "test_video.mp4",
		Mode: 0600,
		Size: int64(len("dummy video data")),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("dummy video data")); err != nil {
		t.Fatal(err)
	}

	// Add a macOS hidden file (should be ignored)
	header = &tar.Header{
		Name: "._test_video.mp4",
		Mode: 0600,
		Size: int64(len("hidden file data")),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("hidden file data")); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	return &buf
}

func TestExtractClipsFromTar(t *testing.T) {
	// Create a test tar file
	tarData := createTestTar(t)

	// Create a temporary file to write the tar data
	tmpFile, err := os.CreateTemp("", "test-*.tar")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(tarData.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Test extracting clips
	clips, err := ExtractClipsFromTar(tmpFile.Name())
	if err != nil {
		t.Fatalf("ExtractClipsFromTar() error = %v", err)
	}

	// Check if we got the expected number of clips
	// Should be 1 because the hidden file should be ignored
	if len(clips) != 1 {
		t.Errorf("ExtractClipsFromTar() got %d clips, want 1", len(clips))
	}

	// Check if the clip has the correct key
	if clips[0].Key != "test_video" {
		t.Errorf("ExtractClipsFromTar() got key %s, want test_video", clips[0].Key)
	}

	// Check if the clip has the correct data
	if string(clips[0].RawData) != "dummy video data" {
		t.Errorf("ExtractClipsFromTar() got data %s, want dummy video data", string(clips[0].RawData))
	}
}
