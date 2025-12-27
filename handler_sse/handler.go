package handlersse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// SSEStats tracks statistics for an SSE endpoint
type SSEStats struct {
	TotalConnections      int64     `json:"totalConnections"`
	CurrentConnections    int       `json:"currentConnections"`
	MessagesSent          int64     `json:"messagesSent"`
	ConnectionsRejected   int64     `json:"connectionsRejected"`
	LastConnectionTime    time.Time `json:"lastConnectionTime"`
	LastDisconnectionTime time.Time `json:"lastDisconnectionTime"`
}

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

// SSEHandler represents a Server-Sent Events endpoint
type SSEHandler struct {
	*handler_role.HandlerRole
	clientManager      *SSEClientManager
	KeepAliveInterval  time.Duration
	OnClientConnect    func(clientID string)
	OnClientDisconnect func(clientID string)
	OnClientRequest    func(req *SSEClientReq)
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(handler handler_role.IHandler) *SSEHandler {
	return &SSEHandler{
		HandlerRole:       &handler_role.HandlerRole{Handler: handler, PathPrefix: "/events"},
		clientManager:     newSSEClientManager(),
		KeepAliveInterval: 15 * time.Second,
	}
}

// SetKeepAliveInterval sets the keep-alive interval for SSE connections
func (sh *SSEHandler) SetKeepAliveInterval(interval time.Duration) *SSEHandler {
	sh.KeepAliveInterval = interval
	return sh
}

// HandleSSE creates an HTTP handler for SSE connections
func (sh *SSEHandler) HandleSSE() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientReq := &SSEClientReq{
			handler:           sh,
			W:                 w,
			R:                 r,
			KeepAliveInterval: int(sh.KeepAliveInterval.Seconds()),
			Metadata:          make(map[string]string),
		}

		// Call custom request handler if set
		if sh.OnClientRequest != nil {
			sh.OnClientRequest(clientReq)
		} else {
			// Default behavior: auto-generate client ID and accept
			clientReq.SetGeneratedClientID("sse")
			clientReq.Accept()
		}
	}
}

// Broadcast sends a message to all connected clients
func (sh *SSEHandler) Broadcast(message string) int {
	return sh.clientManager.broadcast(message)
}

// BroadcastJSON sends a JSON message to all connected clients
func (sh *SSEHandler) BroadcastJSON(data interface{}) (int, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return 0, err
	}
	return sh.clientManager.broadcast(string(jsonData)), nil
}

// SendToClient sends a message to a specific client
func (sh *SSEHandler) SendToClient(clientID string, message string) bool {
	return sh.clientManager.sendToClient(clientID, message)
}

// SendJSONToClient sends a JSON message to a specific client
func (sh *SSEHandler) SendJSONToClient(clientID string, data interface{}) (bool, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}
	return sh.clientManager.sendToClient(clientID, string(jsonData)), nil
}

// GetClientCount returns the number of connected clients
func (sh *SSEHandler) GetClientCount() int {
	return sh.clientManager.getClientCount()
}

// GetStats returns statistics about this SSE endpoint
func (sh *SSEHandler) GetStats() SSEStats {
	return sh.clientManager.getStats()
}

// GetClients returns a list of connected client IDs
func (sh *SSEHandler) GetClients() []string {
	return sh.clientManager.getClients()
}

// Shutdown closes all client connections
func (sh *SSEHandler) Shutdown() {
	sh.clientManager.shutdown()
}

// SSEClientReq represents a client request to connect to an SSE endpoint
type SSEClientReq struct {
	ClientID          string
	handler           *SSEHandler
	W                 http.ResponseWriter
	R                 *http.Request
	KeepAliveInterval int
	Metadata          map[string]string
}

// Accept accepts the client connection and begins streaming events
func (sc *SSEClientReq) Accept() {
	if sc.ClientID == "" {
		sc.ClientID = fmt.Sprintf("sse_%d", time.Now().UnixNano())
	}

	// Set comprehensive SSE headers
	sc.W.Header().Set("Content-Type", "text/event-stream")
	sc.W.Header().Set("Cache-Control", "no-cache, no-transform")
	sc.W.Header().Set("Connection", "keep-alive")
	sc.W.Header().Set("Access-Control-Allow-Origin", "*")
	sc.W.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	sc.W.Header().Set("X-Accel-Buffering", "no")
	sc.W.Header().Set("Transfer-Encoding", "chunked")
	sc.W.Header().Set("Content-Encoding", "identity")

	sc.W.WriteHeader(http.StatusOK)

	// Add this client to the client manager
	clientChan := sc.handler.clientManager.addClient(sc.ClientID)
	defer func() {
		sc.handler.clientManager.removeClient(sc.ClientID)
		if sc.handler.OnClientDisconnect != nil {
			sc.handler.OnClientDisconnect(sc.ClientID)
		}
	}()

	// Notify of connection
	if sc.handler.OnClientConnect != nil {
		sc.handler.OnClientConnect(sc.ClientID)
	}

	// Send initial connection event
	initialPayload := map[string]interface{}{
		"type":     "connected",
		"clientId": sc.ClientID,
	}
	if len(sc.Metadata) > 0 {
		initialPayload["metadata"] = sc.Metadata
	}

	initialData, _ := json.Marshal(initialPayload)
	fmt.Fprintf(sc.W, "event: message\ndata: %s\n\n", initialData)
	if flusher, ok := sc.W.(http.Flusher); ok {
		flusher.Flush()
	} else {
		http.Error(sc.W, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Keep-alive ticker
	keepAliveDuration := 15 * time.Second
	if sc.KeepAliveInterval >= 5 {
		keepAliveDuration = time.Duration(sc.KeepAliveInterval) * time.Second
	}
	keepAliveTicker := time.NewTicker(keepAliveDuration)
	defer keepAliveTicker.Stop()

	ctx := sc.R.Context()
	for {
		select {
		case <-ctx.Done():
			closeMsg := fmt.Sprintf("{\"type\":\"close\",\"reason\":\"context_done\",\"timestamp\":\"%s\"}",
				time.Now().Format(time.RFC3339))
			fmt.Fprintf(sc.W, "event: close\ndata: %s\n\n", closeMsg)
			if flusher, ok := sc.W.(http.Flusher); ok {
				flusher.Flush()
			}
			return
		case <-keepAliveTicker.C:
			keepaliveMsg := fmt.Sprintf("{\"type\":\"keepalive\",\"timestamp\":\"%s\"}",
				time.Now().Format(time.RFC3339))
			fmt.Fprintf(sc.W, "event: keepalive\ndata: %s\n\n", keepaliveMsg)
			if flusher, ok := sc.W.(http.Flusher); ok {
				flusher.Flush()
			}
		case message, ok := <-clientChan:
			if !ok {
				closeMsg := fmt.Sprintf("{\"type\":\"close\",\"reason\":\"channel_closed\",\"timestamp\":\"%s\"}",
					time.Now().Format(time.RFC3339))
				fmt.Fprintf(sc.W, "event: close\ndata: %s\n\n", closeMsg)
				if flusher, ok := sc.W.(http.Flusher); ok {
					flusher.Flush()
				}
				return
			}
			fmt.Fprintf(sc.W, "event: message\ndata: %s\n\n", message)
			if flusher, ok := sc.W.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// Reject rejects the client connection with an error message
func (sc *SSEClientReq) Reject(reason string) {
	if reason == "" {
		reason = "Unauthorized"
	}
	sc.handler.clientManager.stats.ConnectionsRejected++
	http.Error(sc.W, reason, http.StatusUnauthorized)
}

// SetMetadata sets optional metadata for this connection
func (sc *SSEClientReq) SetMetadata(key, value string) {
	if sc.Metadata == nil {
		sc.Metadata = make(map[string]string)
	}
	sc.Metadata[key] = value
}

// SetGeneratedClientID generates a unique client ID
func (sc *SSEClientReq) SetGeneratedClientID(elements ...string) {
	sc.ClientID = fmt.Sprintf("%s_%d", strings.Join(elements, "_"), time.Now().UnixNano())
}
