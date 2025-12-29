package main

import (
	"fmt"
	"log"
	"time"

	wbx "github.com/go-xlite/wbx" // Import to include XAppHandler
	embedfs "github.com/go-xlite/wbx/adapter_fs/embed_fs"
	osfs "github.com/go-xlite/wbx/adapter_fs/os_fs"
	server_data "github.com/go-xlite/wbx/debug/api/server_data"
	debugsse "github.com/go-xlite/wbx/debug/sse"
	client "github.com/go-xlite/wbx/demo/client"
	handlermedia "github.com/go-xlite/wbx/handler_media"
	handlerproxy "github.com/go-xlite/wbx/handler_proxy"
	handlersse "github.com/go-xlite/wbx/handler_sse"
	handlerws "github.com/go-xlite/wbx/handler_ws"
	"github.com/go-xlite/wbx/server/webcast"
	"github.com/go-xlite/wbx/server/webproxy"
	"github.com/go-xlite/wbx/server/websock"
	"github.com/go-xlite/wbx/server/webstream"
	websway "github.com/go-xlite/wbx/server/websway"
	webtrail "github.com/go-xlite/wbx/server/webtrail"
	"github.com/go-xlite/wbx/weblite"
)

type ScopePrefix struct {
	Prefix string
}
type DomainPrefix struct {
	Scope ScopePrefix
}
type HostPrefix struct {
	// this is a prefix for services
}

func main() {
	// Create weblite server using provider
	server := weblite.Provider.Servers.New("demo")
	server.SetPort("8080")
	server.SSL.SetFromFiles("../../certs/cert", "../../certs/priv")
	// Initialize the application with embedded files
	clientInstance := client.NewClient()

	//debugfs.PrintEmbeddedFiles(clientInstance.Content, "[DEBUG] Embedded files:")

	// Create webcast server for SSE connections
	sseServer := webcast.NewWebCast()
	// Create SSE handler
	sseHandler := handlersse.NewSSEHandler(sseServer)
	sseHandler.SetPathPrefix("/xt23/sse")
	server.GetRoutes().HandlePathPrefixFn(sseHandler.PathPrefix.Get(), sseServer.OnRequest)
	sseHandler.Run()
	// Start dummy SSE event streamer
	debugsse.StartDummyStreamer(sseHandler, 3*time.Second)

	// === Webstream (Media Streaming) ===

	// Create filesystem adapter for video data
	videoFsAdapter := osfs.NewOsFsAdapter()
	videoFsAdapter.SetBasePath("../../video_data")

	// Create webstream server
	streamServer := webstream.NewWebStream(videoFsAdapter)
	mediaHandler := handlermedia.NewMediaHandler(streamServer)
	mediaHandler.SetPathPrefix("/xt23/stream")
	server.GetRoutes().ForwardPathPrefixFn(mediaHandler.PathPrefix.Get(), mediaHandler.HandleMedia())

	// === Webproxy (Reverse Proxy) ===
	// Create webproxy server pointing to external service
	proxyServer, _ := webproxy.NewWebproxy("https://file-drop.gtn.one:8080/xt21/")
	proxyHandler := handlerproxy.NewProxyHandler(proxyServer)
	proxyHandler.SetPathPrefix("/xt23/proxy")
	server.GetRoutes().HandlePathPrefixFn(proxyHandler.PathPrefix.Get(), proxyHandler.HandleProxy())

	// === WebSocket Handler ===
	wsServer := websock.NewWebSock()
	wsHandler := handlerws.NewWsHandler(wsServer, "demo-ws")
	wsHandler.SetPathPrefix("/xt23/ws")

	wsServer.OnMessage(func(msg *websock.WsMessage) {
		log.Printf("[WebSocket] Message from %s (%s, session: %s): %s",
			msg.Client.Username, msg.ClientID, msg.SessionID, string(msg.Data))
		// Send message to all clients in the same session (respects Isolated/Shared strategies)
		// Create response message
		response := &websock.WsMessage{
			Client:    msg.Client,
			Data:      []byte(fmt.Sprintf("Echo: %s", string(msg.Data))),
			ClientID:  msg.ClientID,
			SessionID: msg.SessionID,
			SenderID:  msg.SenderID,
		}
		wsServer.SendToSession(response)
	})
	wsHandler.OnConnect = func(client *websock.WsClient) {
		log.Printf("[WebSocket] Client connected: %s (%s, session: %s)", client.Username, client.ID, client.SessionID)
	}
	wsHandler.OnDisconnect = func(client *websock.WsClient) {
		log.Printf("[WebSocket] Client disconnected: %s (%s, session: %s)", client.Username, client.ID, client.SessionID)
	}
	wsHandler.Run()

	// == XApp Handler Setup ===
	sway := websway.NewWebSway()
	xappHandler := wbx.NewXAppHandler(sway)

	embedAdapter := embedfs.NewEmbedFS(&clientInstance.Content)
	embedAdapter.SetBasePath("dist") // Set base path inside the embedded FS
	sway.FsProvider = embedAdapter
	sway.SecurityHeaders = true
	sway.VirtualDirSegment = "p" // Use /p/ for virtual directory

	// Create XApp handler for serving HTML applications

	xappHandler.SetPathPrefix("xt23")
	xappHandler.AuthSkippedPaths = []string{} // No auth for demo

	xappHandler.Run()

	// Initialize server data provider
	serversData := server_data.NewServersDataGen()
	serversData.Initialize(80)

	// Setup API routes
	wbtServersApi := webtrail.NewWebtrail()
	wbtServersApi.GetRoutes().HandlePathFn("/servers/a/list", serversData.HandleListRequest)
	wbtServersApi.GetRoutes().HandlePathFn("/servers/i/{id}/details", serversData.HandleDetailsRequest)
	wbtServersApi.GetRoutes().HandlePathFn("/servers/a/filters", serversData.HandleFiltersRequest)

	apiHandler := wbx.NewApiHandler(wbtServersApi)
	apiHandler.SetPathPrefix("/xt23/trail")
	apiHandler.Run()

	server.GetRoutes().HandlePathPrefixFn("/", sway.OnRequest)
	// Start the server
	log.Println("Server starting on http://localhost:8080")

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
