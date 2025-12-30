package websock

import wsi "github.com/go-xlite/wbx/handler-server/ws"

// WorkerStats represents statistics for a WebSocket worker
type WorkerStats struct {
	CurrentConnections int   `json:"currentConnections"`
	TotalConnections   int64 `json:"totalConnections"`
	MessagesSent       int64 `json:"messagesSent"`
	MessagesReceived   int64 `json:"messagesReceived"`
}

func (ws *WorkerStats) GetCurrentConnections() int {
	return ws.CurrentConnections
}
func (ws *WorkerStats) GetTotalConnections() int64 {
	return ws.TotalConnections
}
func (ws *WorkerStats) GetMessagesSent() int64 {
	return ws.MessagesSent
}
func (ws *WorkerStats) GetMessagesReceived() int64 {
	return ws.MessagesReceived
}

// GetStats returns current statistics
func (ws *WebSock) GetStats() wsi.IWebSocketStats {
	ws.mu.RLock()
	currentConnections := len(ws.clients)
	ws.mu.RUnlock()

	ws.statsMu.RLock()
	stats := &WorkerStats{
		CurrentConnections: currentConnections,
		TotalConnections:   ws.stats.TotalConnections,
		MessagesSent:       ws.stats.MessagesSent,
		MessagesReceived:   ws.stats.MessagesReceived,
	}
	ws.statsMu.RUnlock()

	return stats
}
