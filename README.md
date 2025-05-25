# go-vidprep

A Go-based video preprocessing tool that extracts frames from videos in a tar archive. It processes multiple videos in parallel and supports frame rate adjustment, resizing, consistent frame counts, and WebDataset sharding.

## Features

- Extract frames from video clips in a tar archive
- Support for both JPEG and NumPy output formats
- Consistent frame counts per clip (padding or trimming as needed)
- Parallel processing with configurable number of workers
- WebDataset sharding support for distributed training
- Detailed metadata for each processed clip

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
go-vidprep -tar <input_tar> -out <output_dir> [options]
```

### Options

- `-tar string`: Path to input .tar archive (default "videos.tar")
- `-out string`: Directory to save extracted frames (default "output")
- `-fps int`: Target frames per second (default 8)
- `-size string`: Resize videos to this resolution, e.g. "256x256" (default "256x256")
- `-format string`: Output format (jpg, npy) (default "jpg")
- `-frames int`: Target number of frames per chunk (default 16)
- `-workers int`: Number of parallel workers (default: number of CPU cores)
- `-shard-size int`: Number of chunks per WebDataset shard (default 1000)
- `-shard-dir string`: Output directory for WebDataset shards (optional)

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

Create WebDataset shards from existing processed chunks:
```bash
./govidprep -out processed_frames -shard-dir shards -format jpg
```

## Output Structure

### JPEG Format
```
output/
  video1/
    chunk_00000/
      frame_001.jpg
      frame_002.jpg
      ...
      metadata.json
    chunk_00001/
      ...
  video2/
    ...
```

### NumPy Format
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

### WebDataset Sharding
- Shards are created as tar files containing the specified number of samples
- Each shard is named `shard_XXXXX.tar` where XXXXX is a zero-padded number
- Samples within shards maintain their original filenames
- Sharding is optional and only occurs if `-shard-dir` is specified

### Parameter Relationships
- `frames`: Number of frames per chunk (e.g., 16 frames per chunk)
- `fps`: Frame rate for extraction (e.g., 8 frames per second)
- `shardSize`: Number of chunks per shard (e.g., 1000 chunks per shard)
- Example: With `frames=16`, `fps=8`, and `shardSize=1000`:
  - Each chunk contains 16 frames
  - Each shard contains 1000 chunks
  - Total frames per shard = 16,000 frames
  - Total video duration per shard = 16,000 frames ÷ 8 fps = 2,000 seconds

## Metadata Format

Each clip's metadata is stored in a JSON file with the following structure:

```json
{
  "key": "video1/chunk_00000",
  "fps": 8,
  "frame_count": 16,
  "size": [256, 256],
  "is_padded": false,
  "is_trimmed": false,
  "original_fps": 8
}
```

- `key`: The chunk identifier (original video name + chunk number)
- `fps`: Target frames per second
- `frame_count`: Number of frames in the chunk
- `size`: Frame dimensions [height, width]
- `original_fps`: Original video frame rate

## Important Notes

1. Frame Count Consistency:
   - Each clip will have exactly the target number of frames
   - Clips shorter than the target will be padded with zeros
   - Clips longer than the target will be trimmed
   - Remaining frames that don't form a complete chunk will be discarded

2. WebDataset Sharding:
   - Shards are created as tar files containing the specified number of samples
   - Each shard is named `shard_XXXXX.tar` where XXXXX is a zero-padded number
   - Samples within shards maintain their original filenames
   - Sharding is optional and only occurs if `-shard-dir` is specified

3. Frame Truncation:
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
