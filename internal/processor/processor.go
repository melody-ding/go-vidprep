package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/melody-ding/go-vidprep/internal/types"
	ffmpeg "github.com/u2takey/ffmpeg-go"
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

	// Use ffmpeg-go to process the video
	err = ffmpeg.Input(tempVideoPath).
		Output(filepath.Join(outPath, "frame_%03d.jpg"),
			ffmpeg.KwArgs{
				"vf": ComposeTransforms(transforms...),
			}).
		OverWriteOutput().
		Run()

	return err
}

// ProcessClips processes multiple video clips in parallel
func ProcessClips(clips []types.Clip, outputDir string, fps int, size string, numWorkers int) error {
	if numWorkers <= 0 {
		numWorkers = 4 // Default number of workers
	}

	// Create channels for work distribution and error collection
	jobs := make(chan types.Clip, len(clips))
	errors := make(chan error, len(clips))
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for clip := range jobs {
				if err := ProcessClip(clip, outputDir, fps, size); err != nil {
					errors <- fmt.Errorf("error processing %s: %v", clip.Key, err)
				}
			}
		}()
	}

	// Send jobs to workers
	for _, clip := range clips {
		jobs <- clip
	}
	close(jobs)

	// Wait for all workers to finish
	wg.Wait()
	close(errors)

	// Collect any errors
	var errs []error
	for err := range errors {
		errs = append(errs, err)
	}

	// Return combined errors if any occurred
	if len(errs) > 0 {
		return fmt.Errorf("encountered %d errors: %v", len(errs), errs)
	}
	return nil
}
