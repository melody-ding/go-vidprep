package main

import (
	"flag"
	"fmt"

	"github.com/melody-ding/go-vidprep/internal/processor"
	"github.com/melody-ding/go-vidprep/internal/tar_reader"
)

func main() {
	tarPath := flag.String("tar", "videos.tar", "Path to input .tar archive")
	outputDir := flag.String("out", "output", "Directory to save extracted frames")
	fps := flag.Int("fps", 8, "Target frames per second")
	size := flag.String("size", "256x256", "Resize videos to this resolution (e.g. 256x256)")
	flag.Parse()

	clips, err := tar_reader.ExtractClipsFromTar(*tarPath)
	if err != nil {
		fmt.Printf("Error extracting tar: %v\n", err)
		return
	}

	for _, clip := range clips {
		fmt.Printf("Processing clip: %s\n", clip.Key)
		if err := processor.ProcessClip(clip, *outputDir, *fps, *size); err != nil {
			fmt.Printf("Error processing %s: %v\n", clip.Key, err)
		}
	}
}
