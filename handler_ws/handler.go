package handlerws

import (
	"embed"
	"html/template"
	"net/http"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/handler_ws/worker"
)

//go:embed client/*.html client/*.js
var clientFiles embed.FS

// WebSocketStats represents statistics for a WebSocket handler
type WebSocketStats struct {
	Name               string `json:"name"`
	CurrentConnections int    `json:"currentConnections"`
	TotalConnections   int64  `json:"totalConnections"`
	MessagesSent       int64  `json:"messagesSent"`
	MessagesReceived   int64  `json:"messagesReceived"`
	Route              string `json:"route"`
	IframeRoute        string `json:"iframeRoute"`
	WorkerRoute        string `json:"workerRoute"`
	ManagerRoute       string `json:"managerRoute"`
}

// WsHandler manages WebSocket connections with multiple fallback methods
type WsHandler struct {
	*handler_role.HandlerRole
	Name         string
	WsWorker     *worker.WsWorker
	Route        string
	IframeRoute  string
	WorkerRoute  string
	ManagerRoute string
	OnConnect    func(clientID string, userID int64, username string)
	OnDisconnect func(clientID string, userID int64, username string)
	OnMessage    func(clientID string, userID int64, username string, message []byte)
	GetUserInfo  func(r *http.Request) (username string, userID int64) // Callback to extract user info from request
}

// NewWsHandler creates a new WebSocket handler
func NewWsHandler(handler handler_role.IHandler, name string) *WsHandler {
	wsh := &WsHandler{
		HandlerRole:  &handler_role.HandlerRole{Handler: handler, PathPrefix: "/ws"},
		Name:         name,
		WsWorker:     worker.NewWorker(name),
		Route:        "/connect",
		IframeRoute:  "/iframe",
		WorkerRoute:  "/worker.js",
		ManagerRoute: "/manager.js",
	}

	// Set default user info extractor (returns anonymous user)
	wsh.GetUserInfo = func(r *http.Request) (string, int64) {
		return "anonymous", 0
	}

	return wsh
}

// SetRoutes sets custom routes for the WebSocket handler
func (wsh *WsHandler) SetRoutes(route, iframeRoute, workerRoute, managerRoute string) *WsHandler {
	wsh.Route = route
	wsh.IframeRoute = iframeRoute
	wsh.WorkerRoute = workerRoute
	wsh.ManagerRoute = managerRoute
	return wsh
}

// SetUserInfoExtractor sets the callback to extract user information from requests
func (wsh *WsHandler) SetUserInfoExtractor(fn func(r *http.Request) (username string, userID int64)) *WsHandler {
	wsh.GetUserInfo = fn
	return wsh
}

// Run starts the WebSocket handler and registers all routes
func (wsh *WsHandler) Run() {
	// Start the worker
	go wsh.WsWorker.Run()

	// Register message handler if provided
	if wsh.OnMessage != nil {
		wsh.WsWorker.OnMessage(func(client *worker.WsClient, message []byte) {
			wsh.OnMessage(client.ID, client.UserID, client.Username, message)
		})
	}

	// Register WebSocket connection route
	wsh.Handler.GetRoutes().HandlePathFn(wsh.PathPrefix+wsh.Route, wsh.handleWebSocket)

	// Register iframe route
	wsh.Handler.GetRoutes().HandlePathFn(wsh.PathPrefix+wsh.IframeRoute, wsh.handleIframe)

	// Register worker script route
	wsh.Handler.GetRoutes().HandlePathFn(wsh.PathPrefix+wsh.WorkerRoute, wsh.handleWorkerScript)

	// Register manager script route
	wsh.Handler.GetRoutes().HandlePathFn(wsh.PathPrefix+wsh.ManagerRoute, wsh.handleManagerScript)
}

// handleWebSocket handles WebSocket connection requests
func (wsh *WsHandler) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract user information
	username, userID := wsh.GetUserInfo(r)

	// Get connection ID from query params
	connID := r.URL.Query().Get("connid")

	// Check if this is a cleanup request
	cleanup := r.URL.Query().Get("cleanup")
	if cleanup == "1" {
		wsh.WsWorker.HandleCleanupConnection(w, r, username, userID, connID)
		return
	}

	// Handle the regular WebSocket connection
	wsh.WsWorker.HandleConnection(w, r, username, userID, connID)

	// Call OnConnect callback if set
	if wsh.OnConnect != nil {
		wsh.OnConnect(connID, userID, username)
	}
}

// handleIframe serves the iframe HTML for fallback connections
func (wsh *WsHandler) handleIframe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	data := map[string]interface{}{
		"Route": wsh.PathPrefix + wsh.Route,
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

// handleWorkerScript serves the SharedWorker JavaScript
func (wsh *WsHandler) handleWorkerScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

	data := map[string]interface{}{
		"Route": wsh.PathPrefix + wsh.Route,
	}

	tmplContent, err := clientFiles.ReadFile("client/ browser-shared-worker.js")
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

// handleManagerScript serves the WebSocket manager JavaScript
func (wsh *WsHandler) handleManagerScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

	data := map[string]interface{}{
		"Route":         wsh.PathPrefix + wsh.Route,
		"WsWorkerRoute": wsh.PathPrefix + wsh.WorkerRoute,
		"IframeRoute":   wsh.PathPrefix + wsh.IframeRoute,
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

// GetStats returns statistics for this WebSocket handler
func (wsh *WsHandler) GetStats() WebSocketStats {
	workerStats := wsh.WsWorker.GetStats()

	return WebSocketStats{
		Name:               wsh.Name,
		CurrentConnections: workerStats.CurrentConnections,
		TotalConnections:   workerStats.TotalConnections,
		MessagesSent:       workerStats.MessagesSent,
		MessagesReceived:   workerStats.MessagesReceived,
		Route:              wsh.PathPrefix + wsh.Route,
		IframeRoute:        wsh.PathPrefix + wsh.IframeRoute,
		WorkerRoute:        wsh.PathPrefix + wsh.WorkerRoute,
		ManagerRoute:       wsh.PathPrefix + wsh.ManagerRoute,
	}
}

// SendToUser sends a message to all connections of a specific user
func (wsh *WsHandler) SendToUser(userID int64, message []byte) {
	wsh.WsWorker.SendToUser(userID, message)
}

// SendToClient sends a message to a specific client connection
func (wsh *WsHandler) SendToClient(clientID string, message []byte) bool {
	return wsh.WsWorker.SendToClient(clientID, message)
}

// Broadcast sends a message to all connected clients
func (wsh *WsHandler) Broadcast(message []byte) {
	wsh.WsWorker.Broadcast(message)
}
