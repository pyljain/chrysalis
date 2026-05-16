package localsessions

import "errors"

type Manager interface {
	FetchSessionHistory(sessionId, basePath string) (string, error)
	SetupSession(sessionId, basePath, sessionData string) error
	CopyConfiguration() error
}

var ErrInvalidManager = errors.New("invalid manager")

func GetManager(sessionType string) (Manager, error) {
	switch sessionType {
	case "claude":
		return &Claude{}, nil
	default:
		return nil, ErrInvalidManager
	}
}
