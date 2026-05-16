package storage

import (
	"context"
	"fmt"
)

type Storage interface {
	SaveSessionState(ctx context.Context, sessionId string, sessionHistory string, compressedFiles []byte) error
	GetSessionState(ctx context.Context, sessionId string) (string, []byte, error)
}

func GetStorage(ctx context.Context, storageType string, path string) (Storage, error) {
	switch storageType {
	case "gcs":
		return newGCS(ctx, path)
	case "local":
		return newLocal(path)
	default:
		return nil, fmt.Errorf("storage %s not found", storageType)
	}
}
