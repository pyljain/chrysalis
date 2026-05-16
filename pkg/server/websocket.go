package server

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *Server) WebSocket(w http.ResponseWriter, r *http.Request) {
	sessionId := r.PathValue("id")
	if sessionId == "" {
		http.Error(w, "Session Id must be provided", http.StatusBadRequest)
		return
	}

	// Upgrade to web socket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket", http.StatusBadRequest)
		return
	}

	updateChan := make(chan struct{})
	// TODO handle multiple websockets per session
	s.sessionsMap[sessionId] = updateChan

	// Listen to stream
	for range updateChan {
		// Get latest session from DB
		session, err := s.db.GetSession(sessionId)
		if err != nil {
			log.Printf("Error %s", err)
			break
		}

		err = conn.WriteJSON(session)
		if err != nil {
			log.Printf("Error %s", err)
			break
		}
	}

	delete(s.sessionsMap, sessionId)
}
