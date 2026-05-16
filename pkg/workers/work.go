package workers

import (
	"chrysalis/pkg/bridge"
	"chrysalis/pkg/models"
	"chrysalis/pkg/storage"
	"chrysalis/pkg/utils"
	localsessions "chrysalis/pkg/utils/local_sessions"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const numConcurrentSessionsToHandle = 3

const baseInstructions = `
1. Please remember to create a README for the project.
2. Please create evals to evaluate the agent and score its effectiveness.
`

type Worker struct {
	bridgeConn           bridge.Bridge
	storageClient        storage.Storage
	sessionFilesystemDir string
}

func New(bridgeConn bridge.Bridge, storageClient storage.Storage, sessionFilesystemDir string) *Worker {
	return &Worker{
		bridgeConn:           bridgeConn,
		storageClient:        storageClient,
		sessionFilesystemDir: sessionFilesystemDir,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	workChan := w.bridgeConn.SubscribeForWork(ctx)
	for range numConcurrentSessionsToHandle {
		go func() {
			for work := range workChan {
				log.Printf("Framework is %s", work.Framework)
				localSessionManager, err := localsessions.GetManager("claude")
				if err != nil {
					log.Printf("Invalid local session manager")
					continue
				}

				err = localSessionManager.CopyConfiguration()
				if err != nil {
					log.Printf("Unable to setup the worker to serve sessions with Claude. Error is %s", err)
					continue
				}

				// Record status of Active
				err = w.bridgeConn.RecordSessionStatusUpdate(ctx, work.Id, models.SessionStatusActive)
				if err != nil {
					log.Printf("Unable to update the session's status to Active for session: %s. Error is %s", work.Id, err)
					continue
				}

				// Download session from GCS if exists by sessionID
				sessionHistory, compressedFiles, _ := w.storageClient.GetSessionState(ctx, work.Id)

				sessionPath := filepath.Join(w.sessionFilesystemDir, work.Id)

				err = os.Mkdir(sessionPath, os.ModePerm)
				if err != nil {
					if !errors.Is(err, os.ErrExist) {
						log.Printf("Error creating temp directory %s", err)
						continue
					}
				}

				if sessionHistory != "" {
					// Create Claude session file
					err = localSessionManager.SetupSession(work.Id, sessionPath, sessionHistory)
					if err != nil {
						log.Printf("Error setting up local session %s", err)
						continue
					}

					// Unzip the folder and ensure a root working directory is created, set workdir
					err = utils.Decompress(compressedFiles, sessionPath)
					if err != nil {
						log.Printf("Error decompressing the archive %s", err)
						continue
					}
				}

				// Run claude code for the intent (and/or history) in the session
				logWriter := NewLogWriter(w.bridgeConn, work.Id)

				// TODO: Fetch anything in the session specific queue and add to intent

				intent := work.Intent

				if sessionHistory == "" {
					intent = fmt.Sprintf("<important>\n%s\n%s\n</important>\n<goal>%s</goal>", frameworkMessage(work.Framework), baseInstructions, work.Intent)
				}

				continutation := sessionHistory != ""

				for {
					log.Printf("Starting session %s", work.Id)
					args := []string{"-p", intent, "--dangerously-skip-permissions", "--verbose", "--output-format", "stream-json", "--model", "claude-opus-4-7"}
					if continutation {
						args = append(args, "--resume", work.Id)
					} else {
						args = append(args, "--session-id", work.Id)
					}

					cmd := exec.CommandContext(ctx, "claude", args...)

					// Stream state to frontend
					cmd.Dir = sessionPath
					cmd.Stdout = logWriter
					err = cmd.Run()
					if err != nil {
						log.Printf("Error running claude %s", err)
						continue
					}

					// Record status of Awaiting User Input
					err := w.bridgeConn.RecordSessionStatusUpdate(ctx, work.Id, models.SessionStatusAwaitingUserInput)
					if err != nil {
						log.Printf("Unable to update the session's status to awaiting user input for session: %s. Error is %s", work.Id, err)
						continue
					}

					// Listen on the session queue until timeout
					intent = w.bridgeConn.WatchQueue(ctx, work.Id)
					if intent == "" {
						log.Printf("Timed out on waiting for intent for session %s", work.Id)
						break
					}

					log.Printf("Got new intent %s for session %s", intent, work.Id)
					// Record status of Active
					err = w.bridgeConn.RecordSessionStatusUpdate(ctx, work.Id, models.SessionStatusActive)
					if err != nil {
						log.Printf("Unable to update the session's status to Active for session: %s. Error is %s", work.Id, err)
						continue
					}

					continutation = true

				}

				log.Printf("Timout out. Saving state to GCS")

				// Delete Queue
				// err = w.bridgeConn.DeleteQueue(ctx, work.Id)
				// if err != nil {
				// 	log.Printf("Queue deletion failed with err: %s", err)
				// 	continue
				// }

				// Record status of Active
				err = w.bridgeConn.RecordSessionStatusUpdate(ctx, work.Id, models.SessionStatusInactive)
				if err != nil {
					log.Printf("Unable to update the session's status to Inactive for session: %s. Error is %s", work.Id, err)
					continue
				}

				// Persist session to GCS
				/*
					1. Zip up the code directory

					2. Upload the JSONL file with sesison history
					3. Overwrite GCS path
				*/

				compressedFiles, err = utils.Compress(sessionPath)
				if err != nil {
					log.Printf("Unable to compress session files to save, err: %s", err)
					continue
				}

				// Get session history from the claude session
				sessionHistory, err = localSessionManager.FetchSessionHistory(work.Id, sessionPath)
				if err != nil {
					log.Printf("Unable to get local session history, err: %s", err)
					continue
				}

				err = w.storageClient.SaveSessionState(ctx, work.Id, sessionHistory, compressedFiles)
				if err != nil {
					log.Printf("Unable to save to storage, err: %s", err)
					continue
				}

				// TODO mop up anything that came into the session queue
				// go func ()  {
				// 	time.Sleep(5 * time.Minute)
				// 	// LRANGE on session queue for any messages that have come after inactive
				// 	// Put those messages into the work queue
				// }()

			}
		}()
	}

	<-ctx.Done()

	return nil
}

func frameworkMessage(framework string) string {
	switch framework {
	case "ADK":
		return "You must use Agent Development kit (https://adk.dev/) to build this agent. Please use Javascript for the frontend and python (ADK) in the backend. Refer to the skill `adk-agent` for more information."
	default:
		return ""
	}
}
