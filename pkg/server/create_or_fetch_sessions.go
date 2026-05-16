package server

import (
	"chrysalis/pkg/models"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type createSessionRequest struct {
	Intent    string `json:"intent"`
	Framework string `json:"framework"` // Framework can be ADK or Claude Code SDK
}

func (s *Server) createOrListSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createSession(w, r)

	case http.MethodGet:
		s.listSessions(w, r)
	default:
		http.Error(w, "Unimplemented", http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	creationRequest := createSessionRequest{}

	sessionId, err := uuid.NewV7()
	if err != nil {
		http.Error(w, "Internal Server Error: Unable to generate a Session ID", http.StatusInternalServerError)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&creationRequest)
	if err != nil {
		http.Error(w, "Unable to unmarshal request", http.StatusBadRequest)
		return
	}

	err = s.db.CreateSession(sessionId.String(), creationRequest.Intent, creationRequest.Framework)
	if err != nil {
		http.Error(w, "Unable to create a session", http.StatusInternalServerError)
		return
	}

	session := models.Session{
		Id:        sessionId.String(),
		Intent:    creationRequest.Intent,
		Framework: creationRequest.Framework,
	}

	// err = s.bridgeConn.PublishWork(r.Context(), &session)
	// if err != nil {
	// 	http.Error(w, "Unable to publish work", http.StatusInternalServerError)
	// 	return
	// }

	err = json.NewEncoder(w).Encode(&session)
	if err != nil {
		http.Error(w, "Unable to generate a response", http.StatusInternalServerError)
		return
	}

}

func (s *Server) listSessions(w http.ResponseWriter, _ *http.Request) {
	sessions, err := s.db.GetSessions()
	if err != nil {
		log.Printf("Error while fetching sessions to return to the frontend %s", err)
		http.Error(w, "Unable to get sessions", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(&sessions)
	if err != nil {
		http.Error(w, "Unable to encode sessions", http.StatusInternalServerError)
		return
	}

}
