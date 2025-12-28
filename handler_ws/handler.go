package handlerws

import (
	"net/http"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/websock"
)

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
	websock      *websock.WebSock
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
func NewWsHandler(ws *websock.WebSock, name string) *WsHandler {
	handlerRole := handler_role.NewHandler()
	handlerRole.Handler = ws

	wsh := &WsHandler{
		HandlerRole:  handlerRole,
		Name:         name,
		websock:      ws,
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
	// Start the websock server
	go wsh.websock.Run()

	// Register message handler if provided
	if wsh.OnMessage != nil {
		wsh.websock.OnMessage(func(client *websock.WsClient, message []byte) {
			wsh.OnMessage(client.ID, client.UserID, client.Username, message)
		})
	}

	// Register all client routes through websock server
	wsh.websock.RegisterClientRoutes(
		wsh.PathPrefix.Get(),
		wsh.Route,
		wsh.IframeRoute,
		wsh.WorkerRoute,
		wsh.ManagerRoute,
		wsh.GetUserInfo,
	)
}

// GetStats returns statistics for this WebSocket handler
func (wsh *WsHandler) GetStats() WebSocketStats {
	workerStats := wsh.websock.GetStats()

	return WebSocketStats{
		Name:               wsh.Name,
		CurrentConnections: workerStats.CurrentConnections,
		TotalConnections:   workerStats.TotalConnections,
		MessagesSent:       workerStats.MessagesSent,
		MessagesReceived:   workerStats.MessagesReceived,
		Route:              wsh.PathPrefix.Get() + wsh.Route,
		IframeRoute:        wsh.PathPrefix.Get() + wsh.IframeRoute,
		WorkerRoute:        wsh.PathPrefix.Get() + wsh.WorkerRoute,
		ManagerRoute:       wsh.PathPrefix.Get() + wsh.ManagerRoute,
	}
}

// SendToUser sends a message to all connections of a specific user
func (wsh *WsHandler) SendToUser(userID int64, message []byte) {
	wsh.websock.SendToUser(userID, message)
}

// SendToClient sends a message to a specific client connection
func (wsh *WsHandler) SendToClient(clientID string, message []byte) bool {
	return wsh.websock.SendToClient(clientID, message)
}

// Broadcast sends a message to all connected clients
func (wsh *WsHandler) Broadcast(message []byte) {
	wsh.websock.Broadcast(message)
}
