package sharding

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/melody-ding/go-vidprep/internal/processor"
)

// CreateWebDatasetShards creates WebDataset shards from processed samples
func CreateWebDatasetShards(inputDir, outputDir string, shardSize int, format processor.OutputFormat) error {
	var samples []string
	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		switch format {
		case processor.FormatNPY:
			// For NPY format, collect individual .npy files
			if !info.IsDir() && strings.HasSuffix(path, ".npy") {
				samples = append(samples, path)
			}
		case processor.FormatJPEG:
			// For JPEG format, collect chunk directories containing metadata.json
			if info.IsDir() && strings.Contains(path, "chunk_") {
				if _, err := os.Stat(filepath.Join(path, "metadata.json")); err == nil {
					samples = append(samples, path)
				}
			}
		}
		return nil
	})

	// Create shards
	numShards := (len(samples) + shardSize - 1) / shardSize
	for i := 0; i < numShards; i++ {
		start := i * shardSize
		end := (i + 1) * shardSize
		if end > len(samples) {
			end = len(samples)
		}

		shardPath := filepath.Join(outputDir, fmt.Sprintf("shard_%05d.tar", i))
		if err := createShard(shardPath, samples[start:end], format); err != nil {
			return fmt.Errorf("error creating shard %d: %v", i, err)
		}
	}

	return nil
}

// createShard creates a tar file containing the given samples
func createShard(shardPath string, samples []string, format processor.OutputFormat) error {
	tarFile, err := os.Create(shardPath)
	if err != nil {
		return fmt.Errorf("error creating tar file: %v", err)
	}
	defer tarFile.Close()

	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	for _, sample := range samples {
		if format == processor.FormatNPY {
			// For NPY format, just add the file directly
			data, err := os.ReadFile(sample)
			if err != nil {
				return fmt.Errorf("error reading sample %s: %v", sample, err)
			}

			header := &tar.Header{
				Name: filepath.Base(sample),
				Mode: 0644,
				Size: int64(len(data)),
			}

			if err := tw.WriteHeader(header); err != nil {
				return fmt.Errorf("error writing tar header: %v", err)
			}

			if _, err := tw.Write(data); err != nil {
				return fmt.Errorf("error writing tar data: %v", err)
			}
		} else {
			// For JPEG format, add all files in the chunk directory
			err := filepath.Walk(sample, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					data, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("error reading file %s: %v", path, err)
					}

					// Create relative path within the tar file
					relPath, err := filepath.Rel(filepath.Dir(sample), path)
					if err != nil {
						return fmt.Errorf("error getting relative path: %v", err)
					}
					tarPath := filepath.Join(filepath.Base(sample), relPath)

					header := &tar.Header{
						Name: tarPath,
						Mode: 0644,
						Size: int64(len(data)),
					}

					if err := tw.WriteHeader(header); err != nil {
						return fmt.Errorf("error writing tar header: %v", err)
					}

					if _, err := tw.Write(data); err != nil {
						return fmt.Errorf("error writing tar data: %v", err)
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("error processing chunk directory %s: %v", sample, err)
			}
		}
	}

	return nil
}
