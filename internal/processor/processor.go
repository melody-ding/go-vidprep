package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/melody-ding/go-vidprep/internal/numpy"
	"github.com/melody-ding/go-vidprep/internal/types"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// OutputFormat represents the supported output formats
type OutputFormat string

const (
	FormatJPEG OutputFormat = "jpg"
	FormatNPY  OutputFormat = "npy"
)

// Dimensions represents video frame dimensions
type Dimensions struct {
	Width  int
	Height int
}

// ScaleTransform returns a ScaleTransform with the same dimensions
func (d Dimensions) ScaleTransform() ScaleTransform {
	return ScaleTransform(d)
}

// parseDimensions parses a size string (e.g., "256x256") into width and height
func parseDimensions(size string) (Dimensions, error) {
	dimensions := strings.Split(size, "x")
	if len(dimensions) != 2 {
		return Dimensions{}, fmt.Errorf("invalid size format: %s", size)
	}
	width, err := strconv.Atoi(dimensions[0])
	if err != nil {
		return Dimensions{}, fmt.Errorf("invalid width: %s", dimensions[0])
	}
	height, err := strconv.Atoi(dimensions[1])
	if err != nil {
		return Dimensions{}, fmt.Errorf("invalid height: %s", dimensions[1])
	}
	return Dimensions{Width: width, Height: height}, nil
}

// extractRawFrames extracts raw RGB frames from a video using ffmpeg
func extractRawFrames(videoPath string, dims Dimensions, fps int) ([]byte, error) {
	tempRawPath := filepath.Join(os.TempDir(), filepath.Base(videoPath)+"_raw")
	defer os.Remove(tempRawPath)

	transforms := []Transform{
		FPSTransform{FPS: fps},
		dims.ScaleTransform(),
	}

	err := ffmpeg.Input(videoPath).
		Output(tempRawPath,
			ffmpeg.KwArgs{
				"vf":      ComposeTransforms(transforms...),
				"f":       "rawvideo",
				"pix_fmt": "rgb24",
			}).
		OverWriteOutput().
		Run()
	if err != nil {
		return nil, fmt.Errorf("error extracting raw frames: %v", err)
	}

	rawData, err := os.ReadFile(tempRawPath)
	if err != nil {
		return nil, fmt.Errorf("error reading raw frames: %v", err)
	}

	return rawData, nil
}

// saveNumpyArray saves raw frame data as a NumPy array
func saveNumpyArray(data []byte, dims Dimensions, numFrames int, outputPath string) error {
	// Create the NumPy writer
	writer, err := numpy.NewWriter(outputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	// Write the data with shape (frames, height, width, channels)
	shape := []int{numFrames, dims.Height, dims.Width, 3}
	return writer.Write(data, shape)
}

// saveJPEGFrames saves individual JPEG frames
func saveJPEGFrames(videoPath string, dims Dimensions, fps int, outputPath string) error {
	transforms := []Transform{
		FPSTransform{FPS: fps},
		dims.ScaleTransform(),
	}

	return ffmpeg.Input(videoPath).
		Output(filepath.Join(outputPath, "frame_%03d.jpg"),
			ffmpeg.KwArgs{
				"vf": ComposeTransforms(transforms...),
			}).
		OverWriteOutput().
		Run()
}

// saveMetadata saves clip metadata to a JSON file
func saveMetadata(metadata types.ClipMetadata, outputPath string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %v", err)
	}

	return os.WriteFile(outputPath, data, 0644)
}

// ProcessClip extracts frames from a video clip using ffmpeg
func ProcessClip(clip types.Clip, outputDir string, fps int, size string, format OutputFormat, targetFrames int) error {
	// Create temporary video file
	tempVideoPath := filepath.Join(os.TempDir(), clip.Key+".mp4")
	if err := os.WriteFile(tempVideoPath, clip.RawData, 0644); err != nil {
		return err
	}
	defer os.Remove(tempVideoPath)

	// Parse dimensions
	dims, err := parseDimensions(size)
	if err != nil {
		return err
	}

	// Process based on format
	switch format {
	case FormatNPY:
		// First extract all frames
		outPath := filepath.Join(outputDir, clip.Key)
		if err := os.MkdirAll(outPath, 0755); err != nil {
			return err
		}

		// Extract raw frames
		rawData, err := extractRawFrames(tempVideoPath, dims, fps)
		if err != nil {
			return err
		}

		// Calculate number of frames and chunks
		frameSize := dims.Width * dims.Height * 3
		totalFrames := len(rawData) / frameSize
		numChunks := totalFrames / targetFrames

		// Process each chunk
		for i := 0; i < numChunks; i++ {
			// Extract chunk data
			startFrame := i * targetFrames
			endFrame := (i + 1) * targetFrames
			chunkData := rawData[startFrame*frameSize : endFrame*frameSize]

			// Save as NumPy array
			chunkFile := filepath.Join(outPath, fmt.Sprintf("chunk_%05d.npy", i))
			if err := saveNumpyArray(chunkData, dims, targetFrames, chunkFile); err != nil {
				return err
			}

			// Save metadata for this chunk
			metadata := types.ClipMetadata{
				Key:         fmt.Sprintf("%s/chunk_%05d", clip.Key, i),
				FPS:         fps,
				FrameCount:  targetFrames,
				Size:        []int{dims.Height, dims.Width},
				OriginalFPS: fps,
			}
			metadataFile := filepath.Join(outPath, fmt.Sprintf("chunk_%05d_metadata.json", i))
			if err := saveMetadata(metadata, metadataFile); err != nil {
				return err
			}
		}

		// Handle remaining frames if they form a complete chunk
		remainingFrames := totalFrames % targetFrames
		if remainingFrames == targetFrames {
			startFrame := numChunks * targetFrames
			endFrame := startFrame + targetFrames
			chunkData := rawData[startFrame*frameSize : endFrame*frameSize]

			chunkFile := filepath.Join(outPath, fmt.Sprintf("chunk_%05d.npy", numChunks))
			if err := saveNumpyArray(chunkData, dims, targetFrames, chunkFile); err != nil {
				return err
			}

			metadata := types.ClipMetadata{
				Key:         fmt.Sprintf("%s/chunk_%05d", clip.Key, numChunks),
				FPS:         fps,
				FrameCount:  targetFrames,
				Size:        []int{dims.Height, dims.Width},
				OriginalFPS: fps,
			}
			metadataFile := filepath.Join(outPath, fmt.Sprintf("chunk_%05d_metadata.json", numChunks))
			return saveMetadata(metadata, metadataFile)
		}

		return nil

	default:
		// For JPEG format, first extract all frames
		outPath := filepath.Join(outputDir, clip.Key)
		if err := os.MkdirAll(outPath, 0755); err != nil {
			return err
		}

		// Extract all frames
		err = ffmpeg.Input(tempVideoPath).
			Output(filepath.Join(outPath, "frame_%03d.jpg"),
				ffmpeg.KwArgs{
					"vf": ComposeTransforms(FPSTransform{FPS: fps}, dims.ScaleTransform()),
				}).
			OverWriteOutput().
			Run()
		if err != nil {
			return fmt.Errorf("error extracting frames: %v", err)
		}

		// Get list of extracted frames
		files, err := os.ReadDir(outPath)
		if err != nil {
			return fmt.Errorf("error reading output directory: %v", err)
		}

		// Filter for only jpg files and sort them
		var frameFiles []string
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".jpg") {
				frameFiles = append(frameFiles, file.Name())
			}
		}
		sort.Strings(frameFiles)

		// Calculate number of complete chunks
		totalFrames := len(frameFiles)
		numChunks := totalFrames / targetFrames

		// Process each chunk
		for i := 0; i < numChunks; i++ {
			// Create chunk directory
			chunkDir := filepath.Join(outPath, fmt.Sprintf("chunk_%05d", i))
			if err := os.MkdirAll(chunkDir, 0755); err != nil {
				return err
			}

			// Move frames for this chunk
			startIdx := i * targetFrames
			endIdx := (i + 1) * targetFrames
			for j, frameFile := range frameFiles[startIdx:endIdx] {
				oldPath := filepath.Join(outPath, frameFile)
				newPath := filepath.Join(chunkDir, fmt.Sprintf("frame_%03d.jpg", j+1))
				if err := os.Rename(oldPath, newPath); err != nil {
					return fmt.Errorf("error moving frame %s: %v", frameFile, err)
				}
			}

			// Save metadata for this chunk
			metadata := types.ClipMetadata{
				Key:         fmt.Sprintf("%s/chunk_%05d", clip.Key, i),
				FPS:         fps,
				FrameCount:  targetFrames,
				Size:        []int{dims.Height, dims.Width},
				OriginalFPS: fps,
			}
			if err := saveMetadata(metadata, filepath.Join(chunkDir, "metadata.json")); err != nil {
				return err
			}
		}

		// Handle remaining frames if they form a complete chunk
		remainingFrames := totalFrames % targetFrames
		if remainingFrames == targetFrames {
			chunkDir := filepath.Join(outPath, fmt.Sprintf("chunk_%05d", numChunks))
			if err := os.MkdirAll(chunkDir, 0755); err != nil {
				return err
			}

			startIdx := numChunks * targetFrames
			endIdx := startIdx + targetFrames
			for j, frameFile := range frameFiles[startIdx:endIdx] {
				oldPath := filepath.Join(outPath, frameFile)
				newPath := filepath.Join(chunkDir, fmt.Sprintf("frame_%03d.jpg", j+1))
				if err := os.Rename(oldPath, newPath); err != nil {
					return fmt.Errorf("error moving frame %s: %v", frameFile, err)
				}
			}

			metadata := types.ClipMetadata{
				Key:         fmt.Sprintf("%s/chunk_%05d", clip.Key, numChunks),
				FPS:         fps,
				FrameCount:  targetFrames,
				Size:        []int{dims.Height, dims.Width},
				OriginalFPS: fps,
			}
			return saveMetadata(metadata, filepath.Join(chunkDir, "metadata.json"))
		}

		// Clean up any remaining frames that don't form a complete chunk
		for _, frameFile := range frameFiles[numChunks*targetFrames:] {
			oldPath := filepath.Join(outPath, frameFile)
			if err := os.Remove(oldPath); err != nil {
				return fmt.Errorf("error removing incomplete frame %s: %v", frameFile, err)
			}
		}

		return nil
	}
}

// ProcessClips processes multiple video clips in parallel
func ProcessClips(clips []types.Clip, outputDir string, fps int, size string, format OutputFormat, targetFrames int, numWorkers int) error {
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
				if err := ProcessClip(clip, outputDir, fps, size, format, targetFrames); err != nil {
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
