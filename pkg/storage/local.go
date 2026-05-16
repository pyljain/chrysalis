package storage

import (
	"context"
	"os"
	"path"
)

type local struct {
	path string
}

var _ Storage = (*local)(nil)

func newLocal(path string) (*local, error) {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &local{
		path: path,
	}, nil
}

func (l *local) GetSessionState(ctx context.Context, sessionId string) (string, []byte, error) {
	pathJsonl := path.Join(l.path, sessionId, "session.jsonl")
	pathToFilesZip := path.Join(l.path, sessionId, "files.zip")

	jsonlBytes, err := os.ReadFile(pathJsonl)
	if err != nil {
		return "", nil, err
	}

	filesZipBytes, err := os.ReadFile(pathToFilesZip)
	if err != nil {
		return "", nil, err
	}

	return string(jsonlBytes), filesZipBytes, nil

}

func (l *local) SaveSessionState(ctx context.Context, sessionId string, sessionHistory string, compressedFiles []byte) error {
	dirPath := path.Join(l.path, sessionId)
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		return err
	}

	jsonlPath := path.Join(dirPath, "session.jsonl")
	filesZipPath := path.Join(dirPath, "files.zip")

	err = os.WriteFile(jsonlPath, []byte(sessionHistory), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(filesZipPath, compressedFiles, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
