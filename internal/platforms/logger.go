package platforms

import (
	"fmt"
	"sync"
	"time"
)

var (
	LogMu     sync.Mutex
	LogBuffer []string
)

// Log fügt eine Log-Nachricht mit Zeitstempel zum globalen Buffer hinzu
func Log(format string, a ...interface{}) {
	LogMu.Lock()
	defer LogMu.Unlock()
	msg := fmt.Sprintf(format, a...)
	timestamp := time.Now().Format("15:04:05")
	LogBuffer = append(LogBuffer, fmt.Sprintf("[%s] %s", timestamp, msg))
	if len(LogBuffer) > 200 {
		LogBuffer = LogBuffer[len(LogBuffer)-200:]
	}
}
