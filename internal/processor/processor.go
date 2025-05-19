package processor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/melody-ding/go-vidprep/internal/types"
)

// ProcessClip extracts frames from a video clip using ffmpeg
func ProcessClip(clip types.Clip, outputDir string, fps int, size string) error {
	tempVideoPath := filepath.Join(os.TempDir(), clip.Key+".mp4")
	if err := os.WriteFile(tempVideoPath, clip.RawData, 0644); err != nil {
		return err
	}
	defer os.Remove(tempVideoPath)

	outPath := filepath.Join(outputDir, clip.Key)
	if err := os.MkdirAll(outPath, 0755); err != nil {
		return err
	}

	// Parse size string (e.g., "256x256")
	dimensions := strings.Split(size, "x")
	if len(dimensions) != 2 {
		return fmt.Errorf("invalid size format: %s", size)
	}
	width, err := strconv.Atoi(dimensions[0])
	if err != nil {
		return fmt.Errorf("invalid width: %s", dimensions[0])
	}
	height, err := strconv.Atoi(dimensions[1])
	if err != nil {
		return fmt.Errorf("invalid height: %s", dimensions[1])
	}

	// Create transformations
	transforms := []Transform{
		FPSTransform{FPS: fps},
		ScaleTransform{Width: width, Height: height},
	}

	cmd := exec.Command("ffmpeg",
		"-i", tempVideoPath,
		"-vf", ComposeTransforms(transforms...),
		filepath.Join(outPath, "frame_%03d.jpg"),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
