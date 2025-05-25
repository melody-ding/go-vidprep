# go-vidprep

A Go-based video preprocessing tool that extracts frames from videos in a tar archive. It processes multiple videos in parallel and supports frame rate adjustment and resizing.

## Installation

```bash
# Clone the repository
git clone https://github.com/melody-ding/go-vidprep.git
cd go-vidprep

# Build the binary
go build -o govidprep cmd/govidprep/main.go
```

## Usage

```bash
./govidprep [options]
```

### Options

- `-tar string`: Path to input .tar archive (default "videos.tar")
- `-out string`: Directory to save extracted frames (default "output")
- `-fps int`: Target frames per second (default 8)
- `-size string`: Resize videos to this resolution, e.g. "256x256" (default "256x256")
- `-format string`: Output format (jpg, npy) (default "jpg")
- `-workers int`: Number of parallel workers (default: number of CPU cores)

### Examples

Basic usage with default settings:
```bash
./govidprep -tar my_videos.tar
```

Custom frame rate and resolution:
```bash
./govidprep -tar my_videos.tar -fps 10 -size "512x512"
```

Specify output directory and number of workers:
```bash
./govidprep -tar my_videos.tar -out processed_frames -workers 4
```

Output in NumPy array format:
```bash
./govidprep -tar my_videos.tar -format npy
```

## Output

The tool will:
1. Extract videos from the tar archive
2. Process each video to extract frames at the specified FPS
3. Resize frames to the specified dimensions
4. Save frames in the specified format (jpg or npy) in the output directory

The output structure will be:
```
output/
  video1/
    frames.npy  # For NumPy format - single file containing all frames
    # OR
    frame_001.jpg  # For JPEG format - individual frame files
    frame_002.jpg
    ...
  video2/
    frame_001.jpg
    frame_002.jpg
    ...
```

## Requirements

- Go 1.24 or later
- ffmpeg installed on your system

## Development

### Running Tests

To run the tests, you'll need:
- Go 1.24 or later
- ffmpeg installed on your system

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/processor
go test ./internal/tar_reader
```

The tests use ffmpeg to generate small test videos on-the-fly, so no test video files are included in the repository.

## Notes

- The tool skips macOS hidden files (._*) in the tar archive
- Processing time will be displayed after completion
- Each video's frames are saved in a separate directory named after the video
- For .npy format, the frames are saved as a single NumPy array with shape (frames, height, width, 3)
