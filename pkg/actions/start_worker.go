package actions

import (
	"chrysalis/pkg/bridge"
	"chrysalis/pkg/storage"
	"chrysalis/pkg/workers"
	"context"

	"github.com/urfave/cli/v3"
)

func StartWorker(ctx context.Context, c *cli.Command) error {

	bridgeConnectionString := c.String("bridge-connection-string")
	storageType := c.String("storage-type")
	storagePath := c.String("storage-path")
	sessionFileSystemDir := c.String("session-directory")

	redisBridge, err := bridge.NewRedis(ctx, bridgeConnectionString)
	if err != nil {
		return err
	}

	storage, err := storage.GetStorage(ctx, storageType, storagePath)
	if err != nil {
		return err
	}

	worker := workers.New(redisBridge, storage, sessionFileSystemDir)
	err = worker.Start(ctx)
	if err != nil {
		return err
	}

	return nil
}
