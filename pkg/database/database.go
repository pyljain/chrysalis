package database

import "chrysalis/pkg/models"

type Database interface {
	CreateSession(id string, intent string, framework string) error
	GetSessions() ([]*models.Session, error)
	GetSession(sessionId string) (*models.Session, error)
	AddMessageToHistory(sessionId string, history *models.HistoryItem) error
	UpdateSessionStatus(sessionId string, status models.SessionStatus) error
}
