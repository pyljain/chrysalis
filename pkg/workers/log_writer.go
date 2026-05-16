package workers

import (
	"chrysalis/pkg/bridge"
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

type LogWriter struct {
	bridgeClient bridge.Bridge
	sessionID    string
	buffer       strings.Builder
}

func NewLogWriter(bridgeClient bridge.Bridge, sessionID string) *LogWriter {
	return &LogWriter{
		bridgeClient: bridgeClient,
		sessionID:    sessionID,
	}
}

func (l *LogWriter) Write(p []byte) (n int, err error) {
	// Tee the raw agent stream to stdout so the worker console keeps
	// showing the same output it used to when cmd.Stdout was os.Stdout.
	// Failures writing to stdout are intentionally ignored — they must
	// not interfere with the UI path.
	_, _ = os.Stdout.Write(p)

	l.buffer.Write(p)
	l.processCompleteLines()
	return len(p), nil
}

// processCompleteLines splits the buffer on the LAST newline.
// Everything before that newline is a sequence of complete lines
// (each terminated by '\n') — parse them and drop any that don't
// unmarshal. Everything after the last newline is potentially partial,
// so it stays in the buffer for the next Write.
func (l *LogWriter) processCompleteLines() {
	content := l.buffer.String()
	lastNL := strings.LastIndexByte(content, '\n')
	if lastNL < 0 {
		// No complete line yet — keep buffering.
		return
	}

	complete := content[:lastNL]
	remainder := content[lastNL+1:]

	l.buffer.Reset()
	if remainder != "" {
		l.buffer.WriteString(remainder)
	}

	for line := range strings.SplitSeq(complete, "\n") {
		l.processJSONLine(line)
	}
}

func (l *LogWriter) processJSONLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	u := Update{}
	if err := json.Unmarshal([]byte(line), &u); err != nil {
		// A line between two '\n's that fails to parse is malformed, not partial.
		// Drop it and move on — re-buffering would just poison the next Write.
		log.Printf("LogWriter: skipping malformed line (%s)", err)
		return
	}

	// Only stream assistant-role messages to the UI. Claude Code's
	// stream-json also emits user-role messages whose content carries
	// the tool_result from each tool call — including the entire SKILL.md
	// content returned by the Skill tool. Those are inputs *into* Claude,
	// not outputs from it, and must not appear in the agent timeline.
	if u.Message.Role != "assistant" {
		return
	}

	ctx := context.Background()
	now := time.Now().Format(time.RFC3339)

	for _, c := range u.Message.Content {
		var entry map[string]string
		switch c.Type {
		case "text":
			entry = map[string]string{"time": now, "title": "update", "text": c.Text}
		case "tool_use":
			dataBytes, _ := json.Marshal(c.Input)
			entry = map[string]string{"time": now, "title": "tool", "text": c.Name, "input": string(dataBytes)}
		default:
			continue
		}

		if err := l.bridgeClient.RecordLogLine(ctx, l.sessionID, entry); err != nil {
			// Redis push failure is independent of parsing — log and continue
			// rather than re-buffering a line we already parsed.
			log.Printf("LogWriter: failed to record log line: %v", err)
		}
	}
}

// Flush processes any remaining buffered content as a final line attempt.
// Call this when you're done writing to ensure no data is lost.
func (l *LogWriter) Flush() error {
	remaining := strings.TrimSpace(l.buffer.String())
	l.buffer.Reset()
	if remaining != "" {
		l.processJSONLine(remaining)
	}
	return nil
}
