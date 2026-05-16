package server

import (
	"chrysalis/pkg/bridge"
	"chrysalis/pkg/database"
	"chrysalis/pkg/models"
	"chrysalis/pkg/storage"
	"context"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/sync/errgroup"
)

type Server struct {
	port        int
	db          database.Database
	bridgeConn  bridge.Bridge
	sessionsMap map[string]chan struct{}
	storage     storage.Storage
}

func New(port int, db database.Database, bridgeConn bridge.Bridge, storage storage.Storage) *Server {

	return &Server{
		port:        port,
		db:          db,
		bridgeConn:  bridgeConn,
		sessionsMap: make(map[string]chan struct{}),
		storage:     storage,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/api/v1/sessions", s.createOrListSession)
	http.HandleFunc("/api/v1/sessions/{id}", s.getSessionById)
	http.HandleFunc("/api/v1/sessions/{id}/messages", s.sendMessage)
	http.HandleFunc("/api/v1/sessions/{id}/ws", s.WebSocket)
	http.HandleFunc("/api/v1/sessions/{id}/download", s.DownloadWork)
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	ctx := context.Background()

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		// Listen to redis queue for history and update history
		for logLine := range s.bridgeConn.TrackLogs(egCtx) {
			// Write to database
			sessionId := logLine["sessionId"]
			role := "Tool"
			if logLine["title"] == "update" {
				role = "Assistant"
			}
			err := s.db.AddMessageToHistory(sessionId, &models.HistoryItem{
				Role:    role,
				Content: logLine["text"],
				Input:   logLine["input"],
			})
			if err != nil {
				log.Printf("Error: %s", err)
				continue
			}

			// Publish to session topic
			err = s.bridgeConn.PublishHistoryUpdateNotification(egCtx, sessionId)
			if err != nil {
				log.Printf("Error: %s", err)
				continue
			}
		}

		return nil
	})

	eg.Go(func() error {
		return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
	})

	eg.Go(func() error {
		for sessionId := range s.bridgeConn.SubscribeToHistoryUpdateNotification(egCtx) {
			ch, exists := s.sessionsMap[sessionId]
			if !exists {
				continue
			}
			ch <- struct{}{}
		}

		return nil
	})

	eg.Go(func() error {
		// Subscribe to session status update
		for statusUpdate := range s.bridgeConn.WatchForSessionStatusUpdates(egCtx) {
			err := s.db.UpdateSessionStatus(statusUpdate.SessionID, statusUpdate.Status)
			if err != nil {
				return err
			}

			err = s.bridgeConn.PublishHistoryUpdateNotification(egCtx, statusUpdate.SessionID)
			if err != nil {
				log.Printf("Error: %s", err)
				continue
			}
		}

		return nil
	})

	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
}
