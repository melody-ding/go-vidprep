# go-vidprep

A Go-based video preprocessing tool that extracts frames from videos in a tar archive. It processes multiple videos in parallel and supports frame rate adjustment, resizing, and consistent frame counts.

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
- `-frames int`: Target number of frames per chunk (default 16)
- `-workers int`: Number of parallel workers (default: number of CPU cores)

### Examples

Basic usage with default settings:
```bash
./govidprep -tar my_videos.tar
```

Custom frame rate, resolution, and frame count:
```bash
./govidprep -tar my_videos.tar -fps 10 -size "512x512" -frames 32
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
4. Split each video into chunks of exactly targetFrames length. Trailing frames may be truncated if they do not fit perfectly into a chunk.
5. Save frames in the specified format (jpg or npy) in the output directory
6. Generate metadata for each processed chunk

### Output Structure

#### NumPy Format
```
output/
  video1/
    chunk_00000.npy
    chunk_00000_metadata.json
    chunk_00001.npy
    chunk_00001_metadata.json
    ...
  video2/
    chunk_00000.npy
    chunk_00000_metadata.json
    ...
```

#### JPEG Format
```
output/
  video1/
    chunk_00000/
      frame_001.jpg
      frame_002.jpg
      ...
      metadata.json
    chunk_00001/
      frame_001.jpg
      frame_002.jpg
      ...
      metadata.json
  video2/
    chunk_00000/
      frame_001.jpg
      frame_002.jpg
      ...
      metadata.json
```

### Metadata Format

Each chunk has an associated metadata file with the following information:
```json
{
  "key": "video1/chunk_00000",
  "fps": 8,
  "frame_count": 16,
  "size": [256, 256],
  "original_fps": 8
}
```

- `key`: The chunk identifier (original video name + chunk number)
- `fps`: Target frames per second
- `frame_count`: Number of frames in the chunk
- `size`: Frame dimensions [height, width]
- `original_fps`: Original video frame rate

## Important Notes

### Frame Truncation
- The tool will only save complete chunks of exactly `targetFrames` length
- Any remaining frames that don't form a complete chunk will be discarded
- For example, if a video has 50 frames and `targetFrames` is 16:
  - It will create 3 chunks of 16 frames each (48 frames total)
  - The remaining 2 frames will be discarded
- To avoid losing frames, choose a `targetFrames` value that divides evenly into your expected video lengths

### File Naming
- Chunk numbers use 5 decimal places (00000-99999)
- This supports up to 100,000 chunks per video
- For JPEG format, frame numbers within each chunk use 3 decimal places (001-999)

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
- Each video is split into chunks of exactly targetFrames length
- Each chunk is saved in a separate directory named after the video and chunk number
- For .npy format, each chunk is saved as a single NumPy array with shape (frames, height, width, 3)
- For .jpg format, each chunk is saved as individual frame files
