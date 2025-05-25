package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/melody-ding/go-vidprep/internal/processor"
	"github.com/melody-ding/go-vidprep/internal/tar_reader"
)

func main() {
	tarPath := flag.String("tar", "videos.tar", "Path to input .tar archive")
	outputDir := flag.String("out", "output", "Directory to save extracted frames")
	fps := flag.Int("fps", 8, "Target frames per second")
	size := flag.String("size", "256x256", "Resize videos to this resolution (e.g. 256x256)")
	format := flag.String("format", "jpg", "Output format (jpg, npy)")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of parallel workers (default: number of CPU cores)")
	flag.Parse()

	// Validate format
	outputFormat := processor.OutputFormat(*format)
	switch outputFormat {
	case processor.FormatJPEG, processor.FormatNPY:
		// Valid format
	default:
		fmt.Printf("Error: unsupported format %s. Supported formats are: jpg, npy\n", *format)
		return
	}

	clips, err := tar_reader.ExtractClipsFromTar(*tarPath)
	if err != nil {
		fmt.Printf("Error extracting tar: %v\n", err)
		return
	}

	fmt.Printf("Processing %d clips using %d workers...\n", len(clips), *workers)
	startTime := time.Now()
	if err := processor.ProcessClips(clips, *outputDir, *fps, *size, outputFormat, *workers); err != nil {
		fmt.Printf("Error processing clips: %v\n", err)
		return
	}
	duration := time.Since(startTime)
	fmt.Printf("Processing completed successfully in %v!\n", duration)
}
