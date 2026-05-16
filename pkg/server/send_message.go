package server

import (
	"encoding/json"
	"net/http"
)

type Message struct {
	Content string `json:"content"`
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	sessionId := r.PathValue("id")
	if sessionId == "" {
		http.Error(w, "Session Id must be provided", http.StatusBadRequest)
		return
	}

	msg := Message{}
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		http.Error(w, "Unable to encode session", http.StatusInternalServerError)
		return
	}

	sessionInfo, err := s.db.GetSession(sessionId)
	if err != nil {
		http.Error(w, "Unable to get session from database", http.StatusInternalServerError)
		return
	}

	sessionInfo.Intent = msg.Content

	err = s.bridgeConn.SendMessage(r.Context(), sessionInfo)
	if err != nil {
		http.Error(w, "Unable to send message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
