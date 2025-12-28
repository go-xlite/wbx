package debug_sse

import (
	"fmt"
	"log"
	"time"
)

// Broadcaster interface for SSE message broadcasting
type Broadcaster interface {
	Broadcast(message string) int
}

// StartDummyStreamer starts a dummy SSE event streamer for testing
func StartDummyStreamer(broadcaster Broadcaster, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		counter := 0
		for range ticker.C {
			counter++
			message := fmt.Sprintf("Server event #%d at %s", counter, time.Now().Format("15:04:05"))
			count := broadcaster.Broadcast(message)
			if count > 0 {
				log.Printf("[SSE] Broadcast message to %d clients: %s", count, message)
			}
		}
	}()
}
