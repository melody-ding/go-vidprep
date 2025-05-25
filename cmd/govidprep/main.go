package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/melody-ding/go-vidprep/internal/processor"
	"github.com/melody-ding/go-vidprep/internal/sharding"
	"github.com/melody-ding/go-vidprep/internal/tar_reader"
)

func main() {
	tarPath := flag.String("tar", "", "Path to input .tar archive")
	outputDir := flag.String("out", "output", "Directory to save extracted frames")
	fps := flag.Int("fps", 8, "Target frames per second")
	size := flag.String("size", "256x256", "Resize videos to this resolution (e.g. 256x256)")
	format := flag.String("format", "jpg", "Output format (jpg, npy)")
	targetFrames := flag.Int("frames", 16, "Target number of frames per clip (will pad or trim as needed)")
	workers := flag.Int("workers", runtime.NumCPU(), "Number of parallel workers (default: number of CPU cores)")
	shardSize := flag.Int("shard-size", 1000, "Number of chunks per shard")
	shardDir := flag.String("shard-dir", "", "Output directory for WebDataset shards")
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

	// Check if tar file exists before processing
	if *tarPath != "" {
		if _, err := os.Stat(*tarPath); err == nil {
			// Process the tar file
			clips, err := tar_reader.ExtractClipsFromTar(*tarPath)
			if err != nil {
				fmt.Printf("Error extracting tar: %v\n", err)
				return
			}

			fmt.Printf("Processing %d clips using %d workers...\n", len(clips), *workers)
			startTime := time.Now()
			if err := processor.ProcessClips(clips, *outputDir, *fps, *size, outputFormat, *targetFrames, *workers); err != nil {
				fmt.Printf("Error processing clips: %v\n", err)
				return
			}
			duration := time.Since(startTime)
			fmt.Printf("Processed clips successfully in %v!\n", duration)
		} else {
			fmt.Printf("Skipping clip processing as input file %s does not exist\n", *tarPath)
		}
	} else {
		fmt.Printf("Skipping clip processing as no input file specified\n")
	}

	// Create WebDataset shards if shard directory is specified
	if *shardDir != "" {
		if err := os.MkdirAll(*shardDir, 0755); err != nil {
			fmt.Printf("Error creating shard directory: %v\n", err)
			return
		}
		if err := sharding.CreateWebDatasetShards(*outputDir, *shardDir, *shardSize, outputFormat); err != nil {
			fmt.Printf("Error creating WebDataset shards: %v\n", err)
			return
		}
		fmt.Printf("Created WebDataset shards successfully!\n")
	}
}
