package database

import (
	"chrysalis/pkg/models"
	"database/sql"
	"encoding/json"
	"time"

	_ "turso.tech/database/tursogo"
)

type Turso struct {
	db *sql.DB
}

var _ Database = (*Turso)(nil)

func NewTurso(connectionString string) (*Turso, error) {
	db, err := sql.Open("turso", connectionString)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id VARCHAR(50) PRIMARY KEY,
			intent TEXT NOT NULL,
			created DEFAULT CURRENT_TIMESTAMP,
			status TEXT default 'Pending',
			history_version INTEGER DEFAULT 0,
			history TEXT default '[]',
			framework TEXT
		)
	`)
	if err != nil {
		return nil, err
	}
	return &Turso{
		db: db,
	}, nil
}

func (t *Turso) CreateSession(id string, intent string, framework string) error {
	_, err := t.db.Exec("INSERT INTO sessions (id, intent, framework) VALUES (?, ?, ?)", id, intent, framework)
	if err != nil {
		return err
	}

	return nil
}

func (t *Turso) GetSession(sessionId string) (*models.Session, error) {
	row := t.db.QueryRow("SELECT id, intent, status, history_version, history, framework FROM sessions WHERE id=?", sessionId)

	if row.Err() != nil {
		return nil, row.Err()
	}

	session := models.Session{}
	var history string

	if err := row.Scan(&session.Id, &session.Intent, &session.Status, &session.HistoryVersion, &history, &session.Framework); err != nil {
		return nil, err
	}

	h := []*models.HistoryItem{}
	err := json.Unmarshal([]byte(history), &h)
	if err != nil {
		return nil, err
	}

	session.History = h

	return &session, nil
}

func (t *Turso) GetSessions() ([]*models.Session, error) {
	rows, err := t.db.Query("SELECT id, intent, status, history_version, history, framework FROM sessions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	session := models.Session{}
	sessions := []*models.Session{}

	for rows.Next() {
		var history string
		if err := rows.Scan(&session.Id, &session.Intent, &session.Status, &session.HistoryVersion, &history, &session.Framework); err != nil {
			return nil, err
		}

		h := []*models.HistoryItem{}
		err := json.Unmarshal([]byte(history), &h)
		if err != nil {
			return nil, err
		}

		session.History = h
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

func (t *Turso) AddMessageToHistory(sessionId string, message *models.HistoryItem) error {
	retryCount := 0
	for retryCount < 3 {
		row := t.db.QueryRow("SELECT id, history_version, history, framework FROM sessions WHERE id = ?", sessionId)

		if row.Err() != nil {
			return row.Err()
		}

		session := models.Session{}
		var history string

		if err := row.Scan(&session.Id, &session.HistoryVersion, &history, &session.Framework); err != nil {
			return err
		}

		existingHistory := []*models.HistoryItem{}
		err := json.Unmarshal([]byte(history), &existingHistory)
		if err != nil {
			return err
		}

		existingHistory = append(existingHistory, message)

		historyBytes, err := json.Marshal(existingHistory)
		if err != nil {
			return err
		}

		incrementedVersion := session.HistoryVersion + 1

		res, err := t.db.Exec(`UPDATE SESSIONS 
				SET HISTORY=?, HISTORY_VERSION=?
					WHERE ID = ? AND HISTORY_VERSION = ?`, historyBytes, incrementedVersion, sessionId, session.HistoryVersion)
		if err != nil {
			return err
		}

		count, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if count == 0 {
			retryCount += 1
			time.Sleep(1 * time.Second)
		} else {
			break
		}

	}

	return nil

}

func (t *Turso) UpdateSessionStatus(sessionId string, status models.SessionStatus) error {
	_, err := t.db.Exec(`
		UPDATE SESSIONS 
		SET STATUS=?
		WHERE ID = ?`, status, sessionId)
	if err != nil {
		return err
	}

	return nil
}
