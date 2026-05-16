package localsessions

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type Claude struct{}

var _ Manager = (*Claude)(nil)

func (c *Claude) FetchSessionHistory(sessionId, basePath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Unable to retrieve HOME directory %s", err)
		return "", err
	}

	fp := path.Join(home, ".claude", "projects", encodeClaudePath(basePath), fmt.Sprintf("%s.jsonl", sessionId))
	sessionHistory, err := os.ReadFile(fp)
	if err != nil {
		log.Printf("Unable to retrieve session history for session %s: Error %s", sessionId, err)
		return "", err
	}

	return string(sessionHistory), nil

}

func (c *Claude) SetupSession(sessionId, basePath, sessionData string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Unable to retrieve HOME directory %s", err)
		return err
	}

	fp := path.Join(home, ".claude", "projects", encodeClaudePath(basePath), fmt.Sprintf("%s.jsonl", sessionId))
	os.MkdirAll(filepath.Dir(fp), os.ModePerm)

	err = os.WriteFile(fp, []byte(sessionData), os.ModePerm)
	if err != nil {
		log.Printf("Unable to set session history for session: %s. Error: %s", sessionId, err)
		return err
	}

	return nil

}

func (c *Claude) CopyConfiguration() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	out, err := exec.Command("cp", "-r", "./config/claude/", path.Join(homeDir, ".claude")).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error while setting the worker up %s. Error is %s", out, err)
	}

	return nil
}

func encodeClaudePath(basePath string) string {
	absPath, _ := filepath.Abs(basePath)
	return strings.ReplaceAll(absPath, "/", "-")
}
