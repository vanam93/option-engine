package debuglog

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

const logPath = "debug-31da5f.log"

var mu sync.Mutex

// Write appends one NDJSON debug line for the active session.
func Write(hypothesisID, location, message string, data map[string]any) {
	payload := map[string]any{
		"sessionId":    "31da5f",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	line, err := json.Marshal(payload)
	if err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}
