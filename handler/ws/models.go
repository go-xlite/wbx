package wsh

import (
	"net/http"

	"github.com/go-xlite/wbx/comm/handler_role"
	wsi "github.com/go-xlite/wbx/handler-server/ws"
)

// WebSocketStats represents statistics for a WebSocket handler
type WebSocketStats struct {
	Name               string `json:"name"`
	CurrentConnections int    `json:"currentConnections"`
	TotalConnections   int64  `json:"totalConnections"`
	MessagesSent       int64  `json:"messagesSent"`
	MessagesReceived   int64  `json:"messagesReceived"`
	Route              string `json:"route"`
	WorkerRoute        string `json:"workerRoute"`
	ManagerRoute       string `json:"managerRoute"`
}

type IServerStatsProvider interface {
	GetStats() wsi.IWebSocketStats
}

type ISocketServer interface {
	SendToUser(userID int64, message []byte)
	SendToClient(clientID string, message []byte) bool
	Broadcast(message []byte)
}

type Handler struct {
	*handler_role.HandlerRole
	Name          string
	StatsProvider IServerStatsProvider
	Server        ISocketServer
	Route         string
	WorkerRoute   string
	ManagerRoute  string
	OnConnect     func(clientID string, userID int64, username string)
	OnDisconnect  func(clientID string, userID int64, username string)
	OnMessage     func(clientID string, userID int64, username string, message []byte)
	GetUserInfo   func(r *http.Request) (username string, userID int64) // Callback to extract user info from request
}

// GetStats returns statistics for this WebSocket handler
func (wsh *Handler) GetStats() WebSocketStats {
	workerStats := wsh.StatsProvider.GetStats()

	return WebSocketStats{
		Name:               wsh.Name,
		CurrentConnections: workerStats.GetCurrentConnections(),
		TotalConnections:   workerStats.GetTotalConnections(),
		MessagesSent:       workerStats.GetMessagesSent(),
		MessagesReceived:   workerStats.GetMessagesReceived(),
		Route:              wsh.PathPrefix.Suffix(wsh.Route),
		WorkerRoute:        wsh.PathPrefix.Suffix(wsh.WorkerRoute),
		ManagerRoute:       wsh.PathPrefix.Suffix(wsh.ManagerRoute),
	}
}

func NewHandler(name string) *Handler {
	return &Handler{
		HandlerRole:  handler_role.NewHandler(),
		Name:         name,
		Route:        "/connect",
		WorkerRoute:  "/worker.js",
		ManagerRoute: "/manager.js",
	}
}

// SendToUser sends a message to all connections of a specific user
func (wsh *Handler) SendToUser(userID int64, message []byte) {
	wsh.Server.SendToUser(userID, message)
}

// SendToClient sends a message to a specific client connection
func (wsh *Handler) SendToClient(clientID string, message []byte) bool {
	return wsh.Server.SendToClient(clientID, message)
}

// Broadcast sends a message to all connected clients
func (wsh *Handler) Broadcast(message []byte) {
	wsh.Server.Broadcast(message)
}

// SetUserInfoExtractor sets the callback to extract user information from requests
func (wsh *Handler) SetUserInfoExtractor(fn func(r *http.Request) (username string, userID int64)) {
	wsh.GetUserInfo = fn
}

// SetRoutes sets custom routes for the WebSocket handler
func (wsh *Handler) SetRoutes(route, workerRoute, managerRoute string) {
	wsh.Route = route
	wsh.WorkerRoute = workerRoute
	wsh.ManagerRoute = managerRoute
}
