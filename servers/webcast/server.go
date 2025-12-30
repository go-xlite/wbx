package webcast

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	comm "github.com/go-xlite/wbx/comm"
)

// SSEClientManager handles client connections for a specific SSE endpoint
type SSEClientManager struct {
	clients map[string]chan string
	mutex   sync.RWMutex
	stats   SSEStats
}

func newSSEClientManager() *SSEClientManager {
	return &SSEClientManager{
		clients: make(map[string]chan string),
		stats:   SSEStats{},
	}
}

func (scm *SSEClientManager) addClient(clientID string) chan string {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	client := make(chan string, 10)
	scm.clients[clientID] = client

	scm.stats.TotalConnections++
	scm.stats.CurrentConnections++
	scm.stats.LastConnectionTime = time.Now()

	return client
}

func (scm *SSEClientManager) removeClient(clientID string) {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	if client, exists := scm.clients[clientID]; exists {
		close(client)
		delete(scm.clients, clientID)

		scm.stats.CurrentConnections--
		scm.stats.LastDisconnectionTime = time.Now()
	}
}

func (scm *SSEClientManager) broadcast(message string) int {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	sentCount := 0
	for clientID, client := range scm.clients {
		select {
		case client <- message:
			sentCount++
		default:
			// Client buffer full, remove it asynchronously
			go scm.removeClient(clientID)
		}
	}

	scm.stats.MessagesSent += int64(sentCount)
	return sentCount
}

func (scm *SSEClientManager) sendToClient(clientID string, message string) bool {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	client, exists := scm.clients[clientID]
	if !exists {
		return false
	}

	select {
	case client <- message:
		scm.stats.MessagesSent++
		return true
	default:
		// Client buffer full
		go scm.removeClient(clientID)
		return false
	}
}

func (scm *SSEClientManager) getClientCount() int {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	return len(scm.clients)
}

func (scm *SSEClientManager) getStats() SSEStats {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	return scm.stats
}

func (scm *SSEClientManager) getClients() []string {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	clients := make([]string, 0, len(scm.clients))
	for id := range scm.clients {
		clients = append(clients, id)
	}
	return clients
}

func (scm *SSEClientManager) shutdown() {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	for clientID, client := range scm.clients {
		close(client)
		delete(scm.clients, clientID)
	}

	scm.stats.CurrentConnections = 0
	scm.stats.LastDisconnectionTime = time.Now()
}

func (scm *SSEClientManager) incrementRejections() {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()
	scm.stats.ConnectionsRejected++
}

// WebCast represents a Server-Sent Events (SSE) server for real-time streaming
// Similar to WebTrail but optimized for SSE connections and broadcasting
type WebCast struct {
	*comm.ServerCore
	PathBase      string // Optional base path for convenience (e.g., "/events")
	NotFound      http.HandlerFunc
	clientManager *SSEClientManager
}

// NewWebCast creates a new WebCast instance with proper routing capabilities
func NewWebCast() *WebCast {
	wc := &WebCast{
		ServerCore:    comm.NewServerCore(),
		PathBase:      "",
		clientManager: newSSEClientManager(),
	}
	wc.NotFound = http.NotFound
	return wc
}

// OnRequest handles an incoming HTTP request using the registered routes
// This is the main entry point when the main server forwards a request
func (wc *WebCast) OnRequest(w http.ResponseWriter, r *http.Request) {
	wc.Mux.ServeHTTP(w, r)
}

// MakePath creates a full path by prepending the PathBase (if set)
// Useful for documentation or when you want to know the full proxied path
func (wc *WebCast) MakePath(suffix string) string {
	if wc.PathBase == "" {
		return suffix
	}
	return wc.PathBase + suffix
}

// SetNotFoundHandler sets a custom 404 handler
func (wc *WebCast) SetNotFoundHandler(handler http.HandlerFunc) {
	wc.NotFound = handler
	wc.Mux.NotFoundHandler = handler
}

// Broadcast sends a message to all connected clients
func (wc *WebCast) Broadcast(message string) int {
	return wc.clientManager.broadcast(message)
}

// BroadcastJSON sends a JSON message to all connected clients
func (wc *WebCast) BroadcastJSON(data interface{}) (int, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	return wc.clientManager.broadcast(string(jsonData)), nil
}

// SendToClient sends a message to a specific client
func (wc *WebCast) SendToClient(clientID string, message string) bool {
	return wc.clientManager.sendToClient(clientID, message)
}

// SendJSONToClient sends a JSON message to a specific client
func (wc *WebCast) SendJSONToClient(clientID string, data interface{}) (bool, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}
	return wc.clientManager.sendToClient(clientID, string(jsonData)), nil
}

// GetClientCount returns the number of connected clients
func (wc *WebCast) GetClientCount() int {
	return wc.clientManager.getClientCount()
}

// GetStats returns statistics about this SSE endpoint
func (wc *WebCast) GetStats() SSEStats {
	return wc.clientManager.getStats()
}

// GetClients returns a list of connected client IDs
func (wc *WebCast) GetClients() []string {
	return wc.clientManager.getClients()
}

// Shutdown closes all client connections
func (wc *WebCast) Shutdown() {
	wc.clientManager.shutdown()
}

// AddClient adds a new SSE client connection
func (wc *WebCast) AddClient(clientID string) chan string {
	return wc.clientManager.addClient(clientID)
}

// RemoveClient removes an SSE client connection
func (wc *WebCast) RemoveClient(clientID string) {
	wc.clientManager.removeClient(clientID)
}

// IncrementRejections increments the rejected connections counter
func (wc *WebCast) IncrementRejections() {
	wc.clientManager.incrementRejections()
}

// StreamConfig holds configuration for an SSE stream
type StreamConfig struct {
	ClientID          string
	W                 http.ResponseWriter
	R                 *http.Request
	KeepAliveInterval time.Duration
	Metadata          map[string]string
	OnConnect         func(clientID string)
	OnDisconnect      func(clientID string)
}

// StreamToClient handles the SSE streaming loop for a client
func (wc *WebCast) StreamToClient(config StreamConfig) {
	if config.ClientID == "" {
		config.ClientID = fmt.Sprintf("sse_%d", time.Now().UnixNano())
	}

	// Set comprehensive SSE headers
	config.W.Header().Set("Content-Type", "text/event-stream")
	config.W.Header().Set("Cache-Control", "no-cache, no-transform")
	config.W.Header().Set("Connection", "keep-alive")
	config.W.Header().Set("Access-Control-Allow-Origin", "*")
	config.W.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	config.W.Header().Set("X-Accel-Buffering", "no")
	config.W.Header().Set("Transfer-Encoding", "chunked")
	config.W.Header().Set("Content-Encoding", "identity")

	config.W.WriteHeader(http.StatusOK)

	// Add this client to the client manager
	clientChan := wc.AddClient(config.ClientID)
	defer func() {
		wc.RemoveClient(config.ClientID)
		if config.OnDisconnect != nil {
			config.OnDisconnect(config.ClientID)
		}
	}()

	// Notify of connection
	if config.OnConnect != nil {
		config.OnConnect(config.ClientID)
	}

	// Send initial connection event
	initialPayload := map[string]interface{}{
		"type":     "connected",
		"clientId": config.ClientID,
	}
	if len(config.Metadata) > 0 {
		initialPayload["metadata"] = config.Metadata
	}

	initialData, _ := json.Marshal(initialPayload)
	fmt.Fprintf(config.W, "event: message\ndata: %s\n\n", initialData)
	if flusher, ok := config.W.(http.Flusher); ok {
		flusher.Flush()
	} else {
		http.Error(config.W, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Keep-alive ticker
	keepAliveDuration := 15 * time.Second
	if config.KeepAliveInterval >= 5*time.Second {
		keepAliveDuration = config.KeepAliveInterval
	}
	keepAliveTicker := time.NewTicker(keepAliveDuration)
	defer keepAliveTicker.Stop()

	ctx := config.R.Context()
	for {
		select {
		case <-ctx.Done():
			closeMsg := fmt.Sprintf("{\"type\":\"close\",\"reason\":\"context_done\",\"timestamp\":\"%s\"}",
				time.Now().Format(time.RFC3339))
			fmt.Fprintf(config.W, "event: close\ndata: %s\n\n", closeMsg)
			if flusher, ok := config.W.(http.Flusher); ok {
				flusher.Flush()
			}
			return
		case <-keepAliveTicker.C:
			keepaliveMsg := fmt.Sprintf("{\"type\":\"keepalive\",\"timestamp\":\"%s\"}",
				time.Now().Format(time.RFC3339))
			fmt.Fprintf(config.W, "event: keepalive\ndata: %s\n\n", keepaliveMsg)
			if flusher, ok := config.W.(http.Flusher); ok {
				flusher.Flush()
			}
		case message, ok := <-clientChan:
			if !ok {
				closeMsg := fmt.Sprintf("{\"type\":\"close\",\"reason\":\"channel_closed\",\"timestamp\":\"%s\"}",
					time.Now().Format(time.RFC3339))
				fmt.Fprintf(config.W, "event: close\ndata: %s\n\n", closeMsg)
				if flusher, ok := config.W.(http.Flusher); ok {
					flusher.Flush()
				}
				return
			}
			fmt.Fprintf(config.W, "event: message\ndata: %s\n\n", message)
			if flusher, ok := config.W.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
