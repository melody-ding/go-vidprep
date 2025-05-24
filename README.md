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

## Output

The tool will:
1. Extract videos from the tar archive
2. Process each video to extract frames at the specified FPS
3. Resize frames to the specified dimensions
4. Save frames as numbered JPEG files in the output directory

The output structure will be:
```
output/
  video1/
    frame_001.jpg
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

## Notes

- The tool skips macOS hidden files (._*) in the tar archive
- Processing time will be displayed after completion
- Each video's frames are saved in a separate directory named after the video
