package handlerws

import (
	"net/http"

	wsh "github.com/go-xlite/wbx/handler/ws"
	"github.com/go-xlite/wbx/weblite"
	"github.com/go-xlite/wbx/websock"
)

// WsHandler manages WebSocket connections with multiple fallback methods
type WsHandler struct {
	*wsh.Handler
	websock *websock.WebSock
}

// NewWsHandler creates a new WebSocket handler
func NewWsHandler(ws *websock.WebSock, name string) *WsHandler {

	wsh := &WsHandler{
		Handler: wsh.NewHandler(name),
		websock: ws,
	}
	wsh.Handler.Server = wsh.websock
	wsh.Handler.StatsProvider = ws

	// Set default user info extractor (returns anonymous user)
	wsh.GetUserInfo = func(r *http.Request) (string, int64) {
		return "anonymous", 0
	}
	return wsh
}

// Run starts the WebSocket handler and registers all routes
func (wsh *WsHandler) Run() {
	// Start the websock server
	server := weblite.Provider.Servers.GetByIndex(0)
	if wsh.OnMessage == nil {
		panic("OnMessage callback must be provided for WebSocket handler")
	}
	if server == nil {
		panic("No WebLite server available to register WebSocket handler")
	}

	println("zebra azebra", wsh.PathPrefix.Suffix("ws"))

	server.GetRoutes().HandlePathPrefixFn(wsh.PathPrefix.Suffix("ws"), func(w http.ResponseWriter, r *http.Request) {
		wsh.websock.OnRequest(w, r)
	})
	wsh.websock.PathBase = wsh.PathPrefix.Suffix("ws")
	go wsh.websock.Run()

	// Register message handler if provided
	wsh.websock.OnMessage(func(client *websock.WsClient, message []byte) {
		wsh.OnMessage(client.ID, client.UserID, client.Username, message)
	})

	// Register all client routes through websock server
	wsh.websock.RegisterClientRoutes(
		wsh.Route,
		wsh.IframeRoute,
		wsh.WorkerRoute,
		wsh.ManagerRoute,
		wsh.GetUserInfo,
	)
}
