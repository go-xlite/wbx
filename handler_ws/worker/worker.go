package worker

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WsClient represents a connected WebSocket client
type WsClient struct {
	ID       string
	UserID   int64
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	Worker   *WsWorker
}

// WorkerStats represents statistics for a WebSocket worker
type WorkerStats struct {
	CurrentConnections int   `json:"currentConnections"`
	TotalConnections   int64 `json:"totalConnections"`
	MessagesSent       int64 `json:"messagesSent"`
	MessagesReceived   int64 `json:"messagesReceived"`
}

// WsWorker maintains the set of active clients and broadcasts messages
type WsWorker struct {
	Name string // Name of the worker, useful for logging

	// Registered clients
	clients map[string]*WsClient

	// User to clients mapping for multi-device support
	userClients map[int64]map[string]bool

	// Register requests from clients
	register chan *WsClient

	// Unregister requests from clients
	unregister chan *WsClient

	// Lock for concurrent map access
	mu sync.RWMutex

	// Upgrader for HTTP connections
	upgrader websocket.Upgrader

	// Statistics
	stats   WorkerStats
	statsMu sync.RWMutex

	// Message handler callback
	onMessage func(client *WsClient, message []byte)
}

// NewWorker creates a new WebSocket worker
func NewWorker(name string) *WsWorker {
	return &WsWorker{
		Name:        name,
		clients:     make(map[string]*WsClient),
		userClients: make(map[int64]map[string]bool),
		register:    make(chan *WsClient),
		unregister:  make(chan *WsClient),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		stats: WorkerStats{},
	}
}

// OnMessage sets the message handler callback
func (w *WsWorker) OnMessage(handler func(client *WsClient, message []byte)) *WsWorker {
	w.onMessage = handler
	return w
}

// GetStats returns current statistics for this worker
func (w *WsWorker) GetStats() WorkerStats {
	w.mu.RLock()
	currentConnections := len(w.clients)
	w.mu.RUnlock()

	w.statsMu.RLock()
	stats := WorkerStats{
		CurrentConnections: currentConnections,
		TotalConnections:   w.stats.TotalConnections,
		MessagesSent:       w.stats.MessagesSent,
		MessagesReceived:   w.stats.MessagesReceived,
	}
	w.statsMu.RUnlock()

	return stats
}

// incrementTotalConnections increments the total connections counter
func (w *WsWorker) incrementTotalConnections() {
	w.statsMu.Lock()
	w.stats.TotalConnections++
	w.statsMu.Unlock()
}

// incrementMessagesSent increments the messages sent counter
func (w *WsWorker) incrementMessagesSent() {
	w.statsMu.Lock()
	w.stats.MessagesSent++
	w.statsMu.Unlock()
}

// incrementMessagesReceived increments the messages received counter
func (w *WsWorker) incrementMessagesReceived() {
	w.statsMu.Lock()
	w.stats.MessagesReceived++
	w.statsMu.Unlock()
}

// Run starts the worker processing loop
func (w *WsWorker) Run() {
	for {
		select {
		case client := <-w.register:
			w.mu.Lock()
			w.clients[client.ID] = client

			// Add to user's client list
			if _, ok := w.userClients[client.UserID]; !ok {
				w.userClients[client.UserID] = make(map[string]bool)
			}
			w.userClients[client.UserID][client.ID] = true
			w.mu.Unlock()

			// Increment total connections counter
			w.incrementTotalConnections()

		case client := <-w.unregister:
			w.mu.Lock()
			if _, ok := w.clients[client.ID]; ok {
				// Remove from clients map
				delete(w.clients, client.ID)
				close(client.Send)

				// Remove from user's client list
				if clients, ok := w.userClients[client.UserID]; ok {
					delete(clients, client.ID)
					if len(clients) == 0 {
						delete(w.userClients, client.UserID)
					}
				}
			}
			w.mu.Unlock()
		}
	}
}

// HandleConnection upgrades HTTP connection to WebSocket and manages the client
func (w *WsWorker) HandleConnection(wr http.ResponseWriter, r *http.Request, username string, userID int64, connID string) {
	// Disable compression to prevent "response.Write on hijacked connection" errors
	wr.Header().Set("Content-Encoding", "identity")

	conn, err := w.upgrader.Upgrade(wr, r, nil)
	if err != nil {
		return
	}

	// Generate a connection ID if not provided
	if connID == "" {
		connID = GenerateConnectionID()
	}

	client := &WsClient{
		ID:       connID,
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Worker:   w,
	}

	// Register client
	w.register <- client

	// Start client routines
	go client.readPump()
	go client.writePump()
}

// HandleCleanupConnection handles a cleanup WebSocket connection
func (w *WsWorker) HandleCleanupConnection(wr http.ResponseWriter, r *http.Request, username string, userID int64, connID string) {
	// Disable compression
	wr.Header().Set("Content-Encoding", "identity")

	conn, err := w.upgrader.Upgrade(wr, r, nil)
	if err != nil {
		return
	}

	// Set a short read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read a single message for mode change notification
	_, _, err = conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}

	// Close the connection after receiving the message
	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Cleanup complete"))
	conn.Close()

	// Find and remove any existing connections with this ID
	w.mu.Lock()
	if client, ok := w.clients[connID]; ok {
		delete(w.clients, connID)
		close(client.Send)

		// Remove from user's client list
		if clients, ok := w.userClients[userID]; ok {
			delete(clients, connID)
			if len(clients) == 0 {
				delete(w.userClients, userID)
			}
		}
	}
	w.mu.Unlock()
}

// SendToUser sends a message to all connections of a specific user
func (w *WsWorker) SendToUser(userID int64, message []byte) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if clients, ok := w.userClients[userID]; ok {
		for clientID := range clients {
			if client, ok := w.clients[clientID]; ok {
				select {
				case client.Send <- message:
					w.incrementMessagesSent()
				default:
					close(client.Send)
					delete(w.clients, clientID)
					delete(clients, clientID)
					if len(clients) == 0 {
						delete(w.userClients, userID)
					}
				}
			}
		}
	}
}

// SendToClient sends a message to a specific client connection
func (w *WsWorker) SendToClient(clientID string, message []byte) bool {
	w.mu.RLock()
	client, ok := w.clients[clientID]
	w.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case client.Send <- message:
		w.incrementMessagesSent()
		return true
	default:
		return false
	}
}

// Broadcast sends a message to all connected clients
func (w *WsWorker) Broadcast(message []byte) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, client := range w.clients {
		select {
		case client.Send <- message:
			w.incrementMessagesSent()
		default:
			close(client.Send)
			delete(w.clients, client.ID)
		}
	}
}

// readPump pumps messages from the WebSocket to the worker
func (c *WsClient) readPump() {
	defer func() {
		c.Worker.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log unexpected errors if needed
			}
			break
		}

		// Increment messages received counter
		c.Worker.incrementMessagesReceived()

		// Call message handler if set
		if c.Worker.onMessage != nil {
			c.Worker.onMessage(c, message)
		}
	}
}

// writePump pumps messages from the worker to the WebSocket connection
func (c *WsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GenerateConnectionID creates a unique connection ID
func GenerateConnectionID() string {
	return fmt.Sprintf("%s-%s", time.Now().Format("20060102150405"), RandStringBytes(8))
}

// RandStringBytes generates a random string of n bytes
func RandStringBytes(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
