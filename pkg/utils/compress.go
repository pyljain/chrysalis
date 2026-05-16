package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func Compress(fpath string) ([]byte, error) {
	filesBuffer := &bytes.Buffer{}
	gw := gzip.NewWriter(filesBuffer)
	tw := tar.NewWriter(gw)

	err := filepath.Walk(fpath, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		// Skip anything that isn't a regular file: symlinks (whose Walk-time
		// size is just the link target length but whose opened bytes are the
		// pointed-at file's contents), sockets, FIFOs, devices, etc.
		if !info.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		// Re-stat via the open descriptor — Walk's FileInfo is a snapshot
		// from when the directory was enumerated, and the file may have
		// changed size between then and now.
		fi, err := f.Stat()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		header.Name, err = filepath.Rel(fpath, path)
		if err != nil {
			return err
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Bound the copy to header.Size exactly so we can never overrun.
		// If the file shrank between Stat and Copy, pad with zeros so the
		// tar writer's per-entry byte count matches what the header promised.
		n, copyErr := io.CopyN(tw, f, header.Size)
		if copyErr == io.EOF {
			if pad := header.Size - n; pad > 0 {
				if _, err := tw.Write(make([]byte, pad)); err != nil {
					return err
				}
			}
		} else if copyErr != nil {
			return copyErr
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return filesBuffer.Bytes(), nil
}
