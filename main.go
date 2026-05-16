package main

import (
	"chrysalis/pkg/actions"
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

// ./bin/cy agent start
// ./bin/cy server start
func main() {
	cmd := &cli.Command{
		Name:  "cy",
		Usage: "Build and run agents",
		Commands: []*cli.Command{
			{
				Name:  "worker",
				Usage: "Interact with chrysalis agents",
				Commands: []*cli.Command{
					{
						Name:   "start",
						Usage:  "start the worker",
						Action: actions.StartWorker,
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:  "bridge-connection-string",
								Value: "localhost:6379",
								Usage: "Bridge connectionstring",
							},
							&cli.StringFlag{
								Name:  "storage-type",
								Value: "local",
								Usage: "Type of storage to use for storing session history and files. Can be local or gcs.",
							},
							&cli.StringFlag{
								Name:  "storage-path",
								Value: "./data/storage",
								Usage: "Default storage path for Chrysalis",
							},
							&cli.StringFlag{
								Name:  "session-directory",
								Value: "./temp",
								Usage: "Directory to store session files",
							},
						},
					},
				},
			},
			{
				Name:  "server",
				Usage: "Run the main server",
				Commands: []*cli.Command{
					{
						Name:   "start",
						Usage:  "Start the Chrysalis server",
						Action: actions.StartServer,
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:  "port",
								Value: 9999,
								Usage: "Port at which the server will run",
							},
							&cli.StringFlag{
								Name:  "database",
								Value: "./data/chrysalis.sqlite",
								Usage: "Database name",
							},
							&cli.StringFlag{
								Name:  "bridge-connection-string",
								Value: "localhost:6379",
								Usage: "Bridge connectionstring",
							},
							&cli.StringFlag{
								Name:  "storage-type",
								Value: "local",
								Usage: "Type of storage to use for storing session history and files. Can be local or gcs.",
							},
							&cli.StringFlag{
								Name:  "storage-path",
								Value: "./data/storage",
								Usage: "Default storage path for Chrysalis",
							},
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
