package types

// ClipMetadata represents metadata for a processed video clip
type ClipMetadata struct {
	Key         string `json:"key"`
	FPS         int    `json:"fps"`
	FrameCount  int    `json:"frame_count"`
	Size        []int  `json:"size"`
	IsPadded    bool   `json:"is_padded,omitempty"`
	IsTrimmed   bool   `json:"is_trimmed,omitempty"`
	OriginalFPS int    `json:"original_fps,omitempty"`
}
