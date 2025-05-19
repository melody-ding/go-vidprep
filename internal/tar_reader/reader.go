package tar_reader

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/melody-ding/go-vidprep/internal/types"
)

func ExtractClipsFromTar(tarPath string) ([]types.Clip, error) {
	f, err := os.Open(tarPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tr := tar.NewReader(f)
	var clips []types.Clip

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !strings.HasSuffix(hdr.Name, ".mp4") {
			continue
		}

		key := strings.TrimSuffix(filepath.Base(hdr.Name), ".mp4")
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, tr); err != nil {
			return nil, err
		}

		clips = append(clips, types.Clip{Key: key, RawData: buf.Bytes()})
	}

	return clips, nil
}
