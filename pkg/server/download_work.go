package server

import (
	"log"
	"net/http"
)

func (s *Server) DownloadWork(w http.ResponseWriter, r *http.Request) {
	sessionId := r.PathValue("id")
	if sessionId == "" {
		http.Error(w, "Session Id must be provided", http.StatusBadRequest)
		return
	}

	_, code, err := s.storage.GetSessionState(r.Context(), sessionId)
	if err != nil {
		log.Printf("Error fetching code files to serve %s", err)
		http.Error(w, "Unable to fetch files", http.StatusInternalServerError)
		return
	}

	w.Write(code)

}
