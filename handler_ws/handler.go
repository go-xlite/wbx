package handlerws

import (
	"embed"
	"net/http"
	"strings"

	wsh "github.com/go-xlite/wbx/handler/ws"
	hl1 "github.com/go-xlite/wbx/helpers"
	"github.com/go-xlite/wbx/weblite"
	"github.com/go-xlite/wbx/websock"
)

//go:embed app-dist/*
var efs embed.FS

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
	if server == nil {
		panic("No WebLite server available to register WebSocket handler")
	}

	println("xxxx", wsh.PathPrefix.Suffix("/p"))
	server.GetRoutes().ForwardPathPrefixFn(wsh.PathPrefix.Suffix("p"), func(w http.ResponseWriter, r *http.Request) {
		println("path", r.URL.Path)
		if strings.HasSuffix(r.URL.Path, ".js") {
			data, _ := efs.ReadFile("app-dist" + r.URL.Path)
			hl1.Helpers.WriteJsBytes(w, data)
			return
		}
		hl1.Helpers.WriteNotFound(w)

	})
	server.GetRoutes().ForwardPathPrefixFn(wsh.PathPrefix.Get(), func(w http.ResponseWriter, r *http.Request) {
		wsh.websock.OnRequest(w, r)
	})

	go wsh.websock.Run()

	// Register all client routes through websock server
	wsh.websock.RegisterClientRoutes(
		wsh.EndpointRoute,
		wsh.GetUserInfo,
	)
}
