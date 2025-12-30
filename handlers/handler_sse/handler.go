package handlersse

import (
	"embed"
	"fmt"
	"net/http"
	"strings"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/servers/webcast"
	hl1 "github.com/go-xlite/wbx/utils"
)

//go:embed app-dist/*
var content embed.FS

// SSEHandler represents a Server-Sent Events endpoint
type SSEHandler struct {
	*handler_role.HandlerRole
	webcast            *webcast.WebCast
	KeepAliveInterval  time.Duration
	OnClientConnect    func(clientID string)
	OnClientDisconnect func(clientID string)
	OnClientRequest    func(req *SSEClientReq)
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(wc *webcast.WebCast) *SSEHandler {
	handlerRole := handler_role.NewHandler()
	handlerRole.Handler = wc

	return &SSEHandler{
		HandlerRole:       handlerRole,
		webcast:           wc,
		KeepAliveInterval: 15 * time.Second,
	}
}

// SetKeepAliveInterval sets the keep-alive interval for SSE connections
func (sh *SSEHandler) SetKeepAliveInterval(interval time.Duration) *SSEHandler {
	sh.KeepAliveInterval = interval
	return sh
}

// HandleSSE creates an HTTP handler for SSE connections
func (sh *SSEHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
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

func (sh *SSEHandler) Init() {

	sh.webcast.GetRoutes().ForwardPathPrefixFn(sh.PathPrefix.Suffix("p"), func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			data, _ := content.ReadFile("app-dist" + r.URL.Path)
			hl1.Helpers.WriteJsBytes(w, data)
			return
		}
		hl1.Helpers.WriteNotFound(w)
	})
	sh.webcast.GetRoutes().ForwardPathFn(sh.PathPrefix.Suffix("stream"), sh.HandleSSE)
}

// Broadcast sends a message to all connected clients
func (sh *SSEHandler) Broadcast(message string) int {
	return sh.webcast.Broadcast(message)
}

// BroadcastJSON sends a JSON message to all connected clients
func (sh *SSEHandler) BroadcastJSON(data any) (int, error) {
	return sh.webcast.BroadcastJSON(data)
}

// SendToClient sends a message to a specific client
func (sh *SSEHandler) SendToClient(clientID string, message string) bool {
	return sh.webcast.SendToClient(clientID, message)
}

// SendJSONToClient sends a JSON message to a specific client
func (sh *SSEHandler) SendJSONToClient(clientID string, data any) (bool, error) {
	return sh.webcast.SendJSONToClient(clientID, data)
}

// GetClientCount returns the number of connected clients
func (sh *SSEHandler) GetClientCount() int {
	return sh.webcast.GetClientCount()
}

// GetStats returns statistics about this SSE endpoint
func (sh *SSEHandler) GetStats() webcast.SSEStats {
	return sh.webcast.GetStats()
}

// GetClients returns a list of connected client IDs
func (sh *SSEHandler) GetClients() []string {
	return sh.webcast.GetClients()
}

// Shutdown closes all client connections
func (sh *SSEHandler) Shutdown() {
	sh.webcast.Shutdown()
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
	keepAliveInterval := 15 * time.Second
	if sc.KeepAliveInterval >= 5 {
		keepAliveInterval = time.Duration(sc.KeepAliveInterval) * time.Second
	}

	sc.handler.webcast.StreamToClient(webcast.StreamConfig{
		ClientID:          sc.ClientID,
		W:                 sc.W,
		R:                 sc.R,
		KeepAliveInterval: keepAliveInterval,
		Metadata:          sc.Metadata,
		OnConnect:         sc.handler.OnClientConnect,
		OnDisconnect:      sc.handler.OnClientDisconnect,
	})
}

// Reject rejects the client connection with an error message
func (sc *SSEClientReq) Reject(reason string) {
	if reason == "" {
		reason = "Unauthorized"
	}
	sc.handler.webcast.IncrementRejections()
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
