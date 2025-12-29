package main

import (
	"fmt"
	"log"
	"time"

	wbx "github.com/go-xlite/wbx" // Import to include XAppHandler
	embedfs "github.com/go-xlite/wbx/adapter_fs/embed_fs"
	osfs "github.com/go-xlite/wbx/adapter_fs/os_fs"
	debugsse "github.com/go-xlite/wbx/debug/sse"
	client "github.com/go-xlite/wbx/demo/client"
	handlermedia "github.com/go-xlite/wbx/handler_media"
	handlerproxy "github.com/go-xlite/wbx/handler_proxy"
	handlersse "github.com/go-xlite/wbx/handler_sse"
	handlerws "github.com/go-xlite/wbx/handler_ws"
	"github.com/go-xlite/wbx/webcast"
	"github.com/go-xlite/wbx/weblite"
	"github.com/go-xlite/wbx/webproxy"
	"github.com/go-xlite/wbx/websock"
	"github.com/go-xlite/wbx/webstream"
	websway "github.com/go-xlite/wbx/websway"
)

func main() {
	// Create weblite server using provider
	server := weblite.Provider.Servers.New("demo")
	server.SetPort("8080")
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
	server.GetRoutes().HandlePathPrefixFn("/", sway.OnRequest)

	// Create XApp handler for serving HTML applications

	xappHandler.SetPathPrefix("xt23")
	xappHandler.AuthSkippedPaths = []string{} // No auth for demo

	xappHandler.Run()

	// Start the server
	log.Println("Server starting on http://localhost:8080")

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
