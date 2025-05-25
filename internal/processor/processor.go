package processor

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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

	return os.ReadFile(tempRawPath)
}

// createNumpyHeader creates a NumPy array header with the given shape for NPY v1.0.
// This function returns the full header byte slice, including magic string, version,
// header length, dictionary, and padding.
func createNumpyHeader(shape []int) ([]byte, error) {
	// 1. Construct the Python dictionary literal string
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

	// 2. Calculate padding for the dictionary string
	// The total header length (dict string + 10 bytes for magic+version+length prefix)
	// must be a multiple of 16.
	// The dict string itself needs to be padded so that (len(dictBytes) + 10) is a multiple of 16.
	currentHeaderSize := len(dictBytes) + 10 // 10 = len(magic+version) + len(header_len_prefix)
	padding := (16 - (currentHeaderSize % 16)) % 16
	if padding == 0 && currentHeaderSize%16 != 0 {
		// If currentHeaderSize is already a multiple of 16, padding should be 0.
		// But if currentHeaderSize is 0 (e.g. empty dict), padding should also be 0
		// This edge case ensures we don't add 16 bytes of padding when none is needed.
		// For NPY, a non-zero length means padding will always be non-zero if not a multiple of 16.
		// A common pattern is to ensure total length is *at least* 128 bytes if you want to be flexible.
		// For v1.0, 16-byte alignment is sufficient.
	}

	// 3. Assemble the full header byte slice
	var fullHeader bytes.Buffer

	// Magic string and version (NPY v1.0) - 8 bytes
	fullHeader.Write([]byte{0x93, 'N', 'U', 'M', 'P', 'Y', 0x01, 0x00})

	// Header length (uint16 little-endian) - 2 bytes
	// This is the length of the *dictionary string plus its padding*.
	// The total size of the header block (magic+version+length+dict+padding) must be 16-byte aligned.
	// So the length here is the length of the dictionary string + the padding bytes for it.
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

// saveNumpyArray saves raw frame data as a NumPy array
func saveNumpyArray(data []byte, dims Dimensions, numFrames int, outputPath string) error {
	// NPY arrays are usually (frames, height, width, channels)
	// Channels is 3 for RGB or 1 for grayscale. Assuming 3 channels as per your snippet.
	shape := []int{numFrames, dims.Height, dims.Width, 3}

	// Create the header bytes
	headerBytes, err := createNumpyHeader(shape)
	if err != nil {
		return fmt.Errorf("error creating numpy header: %v", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("error creating npy file: %v", err)
	}
	defer f.Close()

	// Write the complete header (magic string, version, header length, dictionary, padding)
	if _, err := f.Write(headerBytes); err != nil {
		return fmt.Errorf("error writing npy header: %v", err)
	}

	// Write data
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("error writing npy data: %v", err)
	}
	return nil
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

// ProcessClip extracts frames from a video clip using ffmpeg
func ProcessClip(clip types.Clip, outputDir string, fps int, size string, format OutputFormat) error {
	// Create temporary video file
	tempVideoPath := filepath.Join(os.TempDir(), clip.Key+".mp4")
	if err := os.WriteFile(tempVideoPath, clip.RawData, 0644); err != nil {
		return err
	}
	defer os.Remove(tempVideoPath)

	// Create output directory
	outPath := filepath.Join(outputDir, clip.Key)
	if err := os.MkdirAll(outPath, 0755); err != nil {
		return err
	}

	// Parse dimensions
	dims, err := parseDimensions(size)
	if err != nil {
		return err
	}

	// Process based on format
	switch format {
	case FormatNPY:
		// Extract raw frames
		rawData, err := extractRawFrames(tempVideoPath, dims, fps)
		if err != nil {
			return err
		}

		// Calculate number of frames
		frameSize := dims.Width * dims.Height * 3 // 3 channels for RGB
		numFrames := len(rawData) / frameSize

		// Save as NumPy array
		return saveNumpyArray(rawData, dims, numFrames, filepath.Join(outPath, "frames.npy"))

	default:
		// Save as JPEG frames
		return saveJPEGFrames(tempVideoPath, dims, fps, outPath)
	}
}

// ProcessClips processes multiple video clips in parallel
func ProcessClips(clips []types.Clip, outputDir string, fps int, size string, format OutputFormat, numWorkers int) error {
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
				if err := ProcessClip(clip, outputDir, fps, size, format); err != nil {
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
