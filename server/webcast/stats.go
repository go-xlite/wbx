package webcast

import "time"

// SSEStats tracks statistics for an SSE endpoint
type SSEStats struct {
	TotalConnections      int64     `json:"totalConnections"`
	CurrentConnections    int       `json:"currentConnections"`
	MessagesSent          int64     `json:"messagesSent"`
	ConnectionsRejected   int64     `json:"connectionsRejected"`
	LastConnectionTime    time.Time `json:"lastConnectionTime"`
	LastDisconnectionTime time.Time `json:"lastDisconnectionTime"`
}
