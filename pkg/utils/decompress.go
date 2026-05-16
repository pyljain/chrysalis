package utils

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
)

func Decompress(archive []byte, rootPath string) error {

	filesBuffer := bytes.NewBuffer(archive)

	gr, err := gzip.NewReader(filesBuffer)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Printf("Could not read header")
			return err
		}
		log.Printf("Reading header %+v", header)

		target := filepath.Join(rootPath, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			// It's a directory, create it
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			// It's a file, ensure the parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
				return err
			}

			// Create the file
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}

			// Copy data from tar reader to the file
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

			// Restore original file permissions
			os.Chmod(target, os.FileMode(header.Mode))

		}
	}
	return nil
}
