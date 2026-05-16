package storage

import (
	"context"
	"fmt"
	"io"
	"log"

	"cloud.google.com/go/storage"
	"golang.org/x/sync/errgroup"
)

type gcsStorage struct {
	bucketName string
	client     *storage.Client
}

var _ Storage = (*gcsStorage)(nil)

func newGCS(ctx context.Context, bucketName string) (*gcsStorage, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &gcsStorage{
		bucketName: bucketName,
		client:     client,
	}, nil
}

func (s *gcsStorage) createObject(ctx context.Context, name string, contents []byte) error {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(name)
	w := obj.NewWriter(ctx)
	defer w.Close()

	_, err := w.Write(contents)
	if err != nil {
		return err
	}

	return nil
}

func (s *gcsStorage) SaveSessionState(ctx context.Context, sessionId string, sessionHistory string, compressedFiles []byte) error {
	sessionObjectName := fmt.Sprintf("%s/session.jsonl", sessionId)
	compressedFilesObjectName := fmt.Sprintf("%s/files.zip", sessionId)

	fileNameList := []string{sessionObjectName, compressedFilesObjectName}
	contentList := [][]byte{[]byte(sessionHistory), compressedFiles}

	eg, egCtx := errgroup.WithContext(ctx)
	for i, fileName := range fileNameList {
		eg.Go(func() error {
			return s.createObject(egCtx, fileName, contentList[i])
		})
	}

	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (s *gcsStorage) getObject(ctx context.Context, fileName string) ([]byte, error) {
	bucket := s.client.Bucket(s.bucketName)
	obj := bucket.Object(fileName)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	reader.Close()

	dataBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return dataBytes, nil
}

func (s *gcsStorage) GetSessionState(ctx context.Context, sessionId string) (string, []byte, error) {

	sessionObjectName := fmt.Sprintf("%s/session.jsonl", sessionId)
	compressedFilesObjectName := fmt.Sprintf("%s/files.zip", sessionId)
	var sessionStateBytes []byte
	var compressedFilesBytes []byte
	var err error

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		sessionStateBytes, err = s.getObject(egCtx, sessionObjectName)
		if err != nil {
			return err
		}

		return nil
	})

	eg.Go(func() error {
		compressedFilesBytes, err = s.getObject(egCtx, compressedFilesObjectName)
		if err != nil {
			return err
		}

		return nil
	})

	err = eg.Wait()
	if err != nil {
		return "", nil, err
	}

	log.Printf("compressedFilesBytes size %d", len(compressedFilesBytes))
	log.Printf("sessionStateBytes size %d", len(sessionStateBytes))

	return string(sessionStateBytes), compressedFilesBytes, nil
}
