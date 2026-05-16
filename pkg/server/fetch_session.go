package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) getSessionById(w http.ResponseWriter, r *http.Request) {
	sessionId := r.PathValue("id")
	if sessionId == "" {
		http.Error(w, "Session Id must be provided", http.StatusBadRequest)
		return
	}

	session, err := s.db.GetSession(sessionId)
	if err != nil {
		http.Error(w, "Unable to get session", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(&session)
	if err != nil {
		http.Error(w, "Unable to encode session", http.StatusInternalServerError)
		return
	}

}
