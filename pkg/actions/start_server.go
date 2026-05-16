package actions

import (
	"chrysalis/pkg/bridge"
	"chrysalis/pkg/database"
	"chrysalis/pkg/server"
	"chrysalis/pkg/storage"
	"context"

	"github.com/urfave/cli/v3"
)

func StartServer(ctx context.Context, c *cli.Command) error {
	port := c.Int("port")
	dbName := c.String("database")
	bridgeConnectionString := c.String("bridge-connection-string")
	storageType := c.String("storage-type")
	storagePath := c.String("storage-path")

	db, err := database.NewTurso(dbName)
	if err != nil {
		return err
	}

	redisBridge, err := bridge.NewRedis(ctx, bridgeConnectionString)
	if err != nil {
		return err
	}

	storage, err := storage.GetStorage(ctx, storageType, storagePath)
	if err != nil {
		return err
	}

	svr := server.New(port, db, redisBridge, storage)
	err = svr.Start()
	if err != nil {
		return err
	}

	return nil
}
