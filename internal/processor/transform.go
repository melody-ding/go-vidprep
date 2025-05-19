package processor

import (
	"fmt"
	"strings"
)

// Transform represents a video transformation that can be applied using ffmpeg
type Transform interface {
	// FFmpegArgs returns the ffmpeg arguments for this transformation
	FFmpegArgs() []string
}

// FPSTransform sets the output frame rate
type FPSTransform struct {
	FPS int
}

func (t FPSTransform) FFmpegArgs() []string {
	return []string{fmt.Sprintf("fps=%d", t.FPS)}
}

// ScaleTransform resizes the video
type ScaleTransform struct {
	Width  int
	Height int
}

func (t ScaleTransform) FFmpegArgs() []string {
	return []string{fmt.Sprintf("scale=%d:%d", t.Width, t.Height)}
}

// ComposeTransforms combines multiple transformations
func ComposeTransforms(transforms ...Transform) string {
	var args []string
	for _, t := range transforms {
		args = append(args, t.FFmpegArgs()...)
	}
	return strings.Join(args, ",")
}
