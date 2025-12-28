package websock

import (
	"embed"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/go-xlite/wbx/routes"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//go:embed client/*
var clientFiles embed.FS

// WsClient represents a connected WebSocket client
type WsClient struct {
	ID       string
	UserID   int64
	Username string
	Conn     *websocket.Conn
	Send     chan []byte
	WebSock  *WebSock
}

// WorkerStats represents statistics for a WebSocket worker
type WorkerStats struct {
	CurrentConnections int   `json:"currentConnections"`
	TotalConnections   int64 `json:"totalConnections"`
	MessagesSent       int64 `json:"messagesSent"`
	MessagesReceived   int64 `json:"messagesReceived"`
}

// WebSock represents a WebSocket server for real-time bidirectional communication
// Similar to Webcast but for WebSocket connections
type WebSock struct {
	Mux      *mux.Router
	Routes   *routes.Routes
	PathBase string // Optional base path for convenience (e.g., "/ws")
	NotFound http.HandlerFunc

	// WebSocket specific fields
	clients     map[string]*WsClient
	userClients map[int64]map[string]bool
	register    chan *WsClient
	unregister  chan *WsClient
	mu          sync.RWMutex
	upgrader    websocket.Upgrader
	stats       WorkerStats
	statsMu     sync.RWMutex
	onMessage   func(client *WsClient, message []byte)
}

// NewWebSock creates a new WebSock instance with proper routing capabilities
func NewWebSock() *WebSock {
	ws := &WebSock{
		Mux:         mux.NewRouter(),
		PathBase:    "",
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
	ws.Routes = routes.NewRoutes(ws.Mux, 1)
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

// GetRoutes returns the Routes instance
func (ws *WebSock) GetRoutes() *routes.Routes {
	return ws.Routes
}

// GetMux returns the mux.Router instance
func (ws *WebSock) GetMux() *mux.Router {
	return ws.Mux
}

// SetNotFoundHandler sets a custom 404 handler
func (ws *WebSock) SetNotFoundHandler(handler http.HandlerFunc) {
	ws.NotFound = handler
	ws.Mux.NotFoundHandler = handler
}

// OnMessage sets the message handler callback
func (ws *WebSock) OnMessage(handler func(client *WsClient, message []byte)) {
	ws.onMessage = handler
}

// GetStats returns current statistics
func (ws *WebSock) GetStats() WorkerStats {
	ws.mu.RLock()
	currentConnections := len(ws.clients)
	ws.mu.RUnlock()

	ws.statsMu.RLock()
	stats := WorkerStats{
		CurrentConnections: currentConnections,
		TotalConnections:   ws.stats.TotalConnections,
		MessagesSent:       ws.stats.MessagesSent,
		MessagesReceived:   ws.stats.MessagesReceived,
	}
	ws.statsMu.RUnlock()

	return stats
}

// Run starts the WebSocket server processing loop
func (ws *WebSock) Run() {
	for {
		select {
		case client := <-ws.register:
			ws.mu.Lock()
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
			if _, ok := ws.clients[client.ID]; ok {
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

	client := &WsClient{
		ID:       connID,
		UserID:   userID,
		Username: username,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		WebSock:  ws,
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
			c.WebSock.onMessage(c, message)
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

// ServeIframe serves the iframe HTML for fallback connections
func (ws *WebSock) ServeIframe(w http.ResponseWriter, route string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := map[string]interface{}{
		"Route": route,
	}

	tmplContent, err := clientFiles.ReadFile("client/iframe.html")
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("iframe").Parse(string(tmplContent))
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

// ServeWorkerScript serves the SharedWorker JavaScript
func (ws *WebSock) ServeWorkerScript(w http.ResponseWriter, route string) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

	data := map[string]interface{}{
		"Route": route,
	}

	tmplContent, err := clientFiles.ReadFile("client/browser-shared-worker.js")
	if err != nil {
		http.Error(w, "Script not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("worker").Parse(string(tmplContent))
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

// ServeManagerScript serves the WebSocket manager JavaScript
func (ws *WebSock) ServeManagerScript(w http.ResponseWriter, route, wsWorkerRoute, iframeRoute string) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

	data := map[string]interface{}{
		"Route":         route,
		"WsWorkerRoute": wsWorkerRoute,
		"IframeRoute":   iframeRoute,
	}

	tmplContent, err := clientFiles.ReadFile("client/browser-ws-manager.js")
	if err != nil {
		http.Error(w, "Script not found", http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("manager").Parse(string(tmplContent))
	if err != nil {
		http.Error(w, "Template parse error", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

// RegisterClientRoutes registers all client-side routes (iframe, worker, manager scripts)
func (ws *WebSock) RegisterClientRoutes(pathPrefix, connectRoute, iframeRoute, workerRoute, managerRoute string, getUserInfo func(r *http.Request) (username string, userID int64)) {
	fmt.Printf("[WebSock] RegisterClientRoutes - pathPrefix: '%s'\n", pathPrefix)
	fmt.Printf("[WebSock] Registering routes:\n")
	fmt.Printf("  - Connect: %s\n", pathPrefix+connectRoute)
	fmt.Printf("  - Iframe: %s\n", pathPrefix+iframeRoute)
	fmt.Printf("  - Worker: %s\n", pathPrefix+workerRoute)
	fmt.Printf("  - Manager: %s\n", pathPrefix+managerRoute)

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

	// Register iframe route
	ws.Routes.HandlePathFn(pathPrefix+iframeRoute, func(w http.ResponseWriter, r *http.Request) {
		ws.ServeIframe(w, pathPrefix+connectRoute)
	})

	// Register worker script route
	ws.Routes.HandlePathFn(pathPrefix+workerRoute, func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWorkerScript(w, pathPrefix+connectRoute)
	})

	// Register manager script route
	ws.Routes.HandlePathFn(pathPrefix+managerRoute, func(w http.ResponseWriter, r *http.Request) {
		ws.ServeManagerScript(w,
			pathPrefix+connectRoute,
			pathPrefix+workerRoute,
			pathPrefix+iframeRoute,
		)
	})
}
