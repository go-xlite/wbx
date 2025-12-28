package ws

type IWebSocketStats interface {
	GetCurrentConnections() int
	GetTotalConnections() int64
	GetMessagesSent() int64
	GetMessagesReceived() int64
}
