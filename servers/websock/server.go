package websock

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-xlite/wbx/comm"
	"github.com/gorilla/websocket"
)

type WsMessage struct {
	Client    *WsClient
	Data      []byte
	ClientID  string
	SessionID string
	SenderID  string // this assumes the UserId of the sender
}

// WsSession represents a persistent session that survives reconnections
type WsSession struct {
	ID        string
	UserID    int64
	Username  string
	Data      map[string]any
	CreatedAt time.Time
	LastSeen  time.Time
	mu        sync.RWMutex
}

// WsClient represents a connected WebSocket client
type WsClient struct {
	ID        string
	SessionID string
	UserID    int64
	Username  string
	Conn      *websocket.Conn
	Send      chan []byte
	WebSock   *WebSock
}

// WebSock represents a WebSocket server for real-time bidirectional communication
// Similar to Webcast but for WebSocket connections
type WebSock struct {
	*comm.ServerCore
	PathBase string // Optional base path for convenience (e.g., "/ws")
	NotFound http.HandlerFunc

	// WebSocket specific fields
	clients     map[string]*WsClient
	userClients map[int64]map[string]bool
	sessions    map[string]*WsSession
	register    chan *WsClient
	unregister  chan *WsClient
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
	stats       WorkerStats
	statsMu     sync.RWMutex
	onMessage   func(msg *WsMessage)
}

// NewWebSock creates a new WebSock instance with proper routing capabilities
func NewWebSock() *WebSock {
	ws := &WebSock{
		ServerCore:  comm.NewServerCore(),
		clients:     make(map[string]*WsClient),
		userClients: make(map[int64]map[string]bool),
		sessions:    make(map[string]*WsSession),
		register:    make(chan *WsClient),
		unregister:  make(chan *WsClient),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
		stats: WorkerStats{},
	}
	ws.NotFound = http.NotFound
	return ws
}

// OnRequest handles an incoming HTTP request using the registered routes
func (ws *WebSock) OnRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[WebSock] OnRequest: %s %s\n", r.Method, r.URL.Path)
	ws.Mux.ServeHTTP(w, r)
}

// MakePath creates a full path by prepending the PathBase (if set)
func (ws *WebSock) MakePath(suffix string) string {
	if ws.PathBase == "" {
		return suffix
	}
	return ws.PathBase + suffix
}

// SetNotFoundHandler sets a custom 404 handler
func (ws *WebSock) SetNotFoundHandler(handler http.HandlerFunc) {
	ws.NotFound = handler
	ws.Mux.NotFoundHandler = handler
}

// OnMessage sets the message handler callback
func (ws *WebSock) OnMessage(handler func(msg *WsMessage)) {
	ws.onMessage = handler

}

// Run starts the WebSocket server processing loop
func (ws *WebSock) Run() {
	for {
		select {
		case client := <-ws.register:
			ws.mu.Lock()

			// If a client with this ID already exists, close it first
			if existingClient, exists := ws.clients[client.ID]; exists {
				fmt.Printf("[WebSock] Duplicate connection ID detected: %s - closing old connection\n", client.ID)

				// Remove the old client from maps BEFORE closing to prevent unregister from affecting new client
				delete(ws.clients, existingClient.ID)
				if clients, ok := ws.userClients[existingClient.UserID]; ok {
					delete(clients, existingClient.ID)
					if len(clients) == 0 {
						delete(ws.userClients, existingClient.UserID)
					}
				}

				// Close the old connection in the background
				// Don't close the channel here - let readPump->unregister handle it
				go func(c *WsClient) {
					c.Conn.Close()
				}(existingClient)
			}

			ws.clients[client.ID] = client

			if _, ok := ws.userClients[client.UserID]; !ok {
				ws.userClients[client.UserID] = make(map[string]bool)
			}
			ws.userClients[client.UserID][client.ID] = true
			ws.mu.Unlock()

			ws.statsMu.Lock()
			ws.stats.TotalConnections++
			ws.statsMu.Unlock()

		case client := <-ws.unregister:
			ws.mu.Lock()
			// Only delete if this exact client instance is still in the map
			// (prevents deleting a newer client with the same ID)
			if existingClient, ok := ws.clients[client.ID]; ok && existingClient == client {
				delete(ws.clients, client.ID)
				close(client.Send)

				if clients, ok := ws.userClients[client.UserID]; ok {
					delete(clients, client.ID)
					if len(clients) == 0 {
						delete(ws.userClients, client.UserID)
					}
				}
			}
			ws.mu.Unlock()
		}
	}
}

// HandleConnection upgrades HTTP connection to WebSocket and manages the client
func (ws *WebSock) HandleConnection(wr http.ResponseWriter, r *http.Request, username string, userID int64, connID string) {
	wr.Header().Set("Content-Encoding", "identity")

	conn, err := ws.upgrader.Upgrade(wr, r, nil)
	if err != nil {
		return
	}

	if connID == "" {
		connID = GenerateConnectionID()
	}

	// Get or create session
	sessionID := r.URL.Query().Get("sessionid")
	if sessionID == "" {
		sessionID = GenerateConnectionID()
	}

	_ = ws.GetOrCreateSession(sessionID, userID, username)

	client := &WsClient{
		ID:        connID,
		SessionID: sessionID,
		UserID:    userID,
		Username:  username,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		WebSock:   ws,
	}

	ws.register <- client

	go client.readPump()
	go client.writePump()
}

// HandleCleanupConnection handles a cleanup WebSocket connection
func (ws *WebSock) HandleCleanupConnection(wr http.ResponseWriter, r *http.Request, username string, userID int64, connID string) {
	wr.Header().Set("Content-Encoding", "identity")

	conn, err := ws.upgrader.Upgrade(wr, r, nil)
	if err != nil {
		return
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	_, _, err = conn.ReadMessage()
	if err != nil {
		conn.Close()
		return
	}

	conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Cleanup complete"))
	conn.Close()

	ws.mu.Lock()
	if client, ok := ws.clients[connID]; ok {
		delete(ws.clients, connID)
		close(client.Send)

		if clients, ok := ws.userClients[userID]; ok {
			delete(clients, connID)
			if len(clients) == 0 {
				delete(ws.userClients, userID)
			}
		}
	}
	ws.mu.Unlock()
}

// SendToUser sends a message to all connections of a specific user
func (ws *WebSock) SendToUser(userID int64, message []byte) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	if clients, ok := ws.userClients[userID]; ok {
		for clientID := range clients {
			if client, ok := ws.clients[clientID]; ok {
				select {
				case client.Send <- message:
					ws.incrementMessagesSent()
				default:
					close(client.Send)
					delete(ws.clients, clientID)
					delete(clients, clientID)
					if len(clients) == 0 {
						delete(ws.userClients, userID)
					}
				}
			}
		}
	}
}

// SendToClient sends a message to a specific client connection
func (ws *WebSock) SendToClient(clientID string, message []byte) bool {
	ws.mu.RLock()
	client, ok := ws.clients[clientID]
	ws.mu.RUnlock()

	if !ok {
		return false
	}

	select {
	case client.Send <- message:
		ws.incrementMessagesSent()
		return true
	default:
		return false
	}
}

// Broadcast sends a message to all connected clients
func (ws *WebSock) Broadcast(message []byte) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	for _, client := range ws.clients {
		select {
		case client.Send <- message:
			ws.incrementMessagesSent()
		default:
			close(client.Send)
			delete(ws.clients, client.ID)
		}
	}
}

// GetOrCreateSession gets an existing session or creates a new one
func (ws *WebSock) GetOrCreateSession(sessionID string, userID int64, username string) *WsSession {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	session, exists := ws.sessions[sessionID]
	if !exists {
		session = &WsSession{
			ID:        sessionID,
			UserID:    userID,
			Username:  username,
			Data:      make(map[string]any),
			CreatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		ws.sessions[sessionID] = session
	} else {
		session.LastSeen = time.Now()
	}

	return session
}

// GetSession retrieves a session by ID
func (ws *WebSock) GetSession(sessionID string) (*WsSession, bool) {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	session, exists := ws.sessions[sessionID]
	return session, exists
}

// DeleteSession removes a session
func (ws *WebSock) DeleteSession(sessionID string) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	delete(ws.sessions, sessionID)
}

// SetSessionData sets a value in the session data
func (session *WsSession) Set(key string, value any) {
	session.mu.Lock()
	defer session.mu.Unlock()
	session.Data[key] = value
}

// GetSessionData gets a value from the session data
func (session *WsSession) Get(key string) (any, bool) {
	session.mu.RLock()
	defer session.mu.RUnlock()
	value, exists := session.Data[key]
	return value, exists
}

// DeleteSessionData deletes a key from the session data
func (session *WsSession) Delete(key string) {
	session.mu.Lock()
	defer session.mu.Unlock()
	delete(session.Data, key)
}

// SendToSession sends a message to all clients in a session
func (ws *WebSock) SendToSession(msg *WsMessage) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	sent := false
	for _, client := range ws.clients {
		if client.SessionID == msg.SessionID {
			select {
			case client.Send <- msg.Data:
				ws.incrementMessagesSent()
				sent = true
			default:
				// Client buffer full, skip
			}
		}
	}
	return sent
}

// SendToSessionExcept sends a message to all clients in a session EXCEPT the specified client
// Useful for broadcasting updates without echoing back to the sender
func (ws *WebSock) SendToSessionExcept(msg *WsMessage, excludeClientID string) bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	sent := false
	for _, client := range ws.clients {
		if client.SessionID == msg.SessionID && client.ID != excludeClientID {
			select {
			case client.Send <- msg.Data:
				ws.incrementMessagesSent()
				sent = true
			default:
				// Client buffer full, skip
			}
		}
	}
	return sent
}

// GetSessionClients returns all clients connected to a session
func (ws *WebSock) GetSessionClients(sessionID string) []*WsClient {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	var clients []*WsClient
	for _, client := range ws.clients {
		if client.SessionID == sessionID {
			clients = append(clients, client)
		}
	}
	return clients
}

// GetSessionConnectionCount returns the number of active connections for a session
func (ws *WebSock) GetSessionConnectionCount(sessionID string) int {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	count := 0
	for _, client := range ws.clients {
		if client.SessionID == sessionID {
			count++
		}
	}
	return count
}

func (ws *WebSock) incrementMessagesSent() {
	ws.statsMu.Lock()
	ws.stats.MessagesSent++
	ws.statsMu.Unlock()
}

func (ws *WebSock) incrementMessagesReceived() {
	ws.statsMu.Lock()
	ws.stats.MessagesReceived++
	ws.statsMu.Unlock()
}

// readPump pumps messages from the WebSocket to the server
func (c *WsClient) readPump() {
	defer func() {
		c.WebSock.unregister <- c
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

		c.WebSock.incrementMessagesReceived()

		if c.WebSock.onMessage != nil {
			msg := &WsMessage{
				Client:    c,
				Data:      message,
				ClientID:  c.ID,
				SessionID: c.SessionID,
				SenderID:  fmt.Sprintf("%d", c.UserID),
			}
			c.WebSock.onMessage(msg)
		}
	}
}

// writePump pumps messages from the server to the WebSocket connection
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

// RegisterClientRoutes registers all client-side routes (worker, manager scripts)
func (ws *WebSock) RegisterClientRoutes(connectRoute string, getUserInfo func(r *http.Request) (username string, userID int64)) {
	pathPrefix := ws.PathBase
	fmt.Printf("[WebSock] RegisterClientRoutes - pathPrefix: '%s'\n", pathPrefix)
	fmt.Printf("[WebSock] Registering routes:\n")
	fmt.Printf("  - Connect: %s\n", pathPrefix+connectRoute)

	// Register WebSocket connection route
	ws.Routes.HandlePathFn(pathPrefix+connectRoute, func(w http.ResponseWriter, r *http.Request) {
		username, userID := getUserInfo(r)
		connID := r.URL.Query().Get("connid")
		cleanup := r.URL.Query().Get("cleanup")

		if cleanup == "1" {
			ws.HandleCleanupConnection(w, r, username, userID, connID)
			return
		}

		ws.HandleConnection(w, r, username, userID, connID)
	})

}
