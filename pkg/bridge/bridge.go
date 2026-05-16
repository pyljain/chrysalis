package bridge

import (
	"chrysalis/pkg/models"
	"context"
)

type Bridge interface {
	PublishWork(ctx context.Context, session *models.Session) error
	SubscribeForWork(ctx context.Context) chan models.Session
	// PublishSessionUpdates(ctx context.Context, update *SessionUpdate) error
	// SubscribeForSessionUpdates(ctx context.Context, sessionID string) (chan SessionUpdate, error)

	SendMessage(ctx context.Context, session *models.Session) error
	WatchQueue(ctx context.Context, sessionId string) string
	DeleteQueue(ctx context.Context, sessionId string) error

	RecordLogLine(ctx context.Context, sessionId string, logLine map[string]string) error
	TrackLogs(ctx context.Context) chan map[string]string
	PublishHistoryUpdateNotification(ctx context.Context, sessionId string) error
	SubscribeToHistoryUpdateNotification(ctx context.Context) chan string

	RecordSessionStatusUpdate(ctx context.Context, sessionId string, status models.SessionStatus) error
	WatchForSessionStatusUpdates(ctx context.Context) chan SessionStatusUpdate
}

type SessionStatusUpdate struct {
	SessionID string               `json:"sessionID"`
	Status    models.SessionStatus `json:"status"`
}
