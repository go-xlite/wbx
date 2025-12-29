package main

import (
	"fmt"
	"log"
	"time"

	wbx "github.com/go-xlite/wbx" // Import to include XAppHandler
	embedfs "github.com/go-xlite/wbx/adapter_fs/embed_fs"
	osfs "github.com/go-xlite/wbx/adapter_fs/os_fs"
	debugfs "github.com/go-xlite/wbx/debug/fs"
	debugsse "github.com/go-xlite/wbx/debug/sse"
	"github.com/go-xlite/wbx/demo/app"
	handlermedia "github.com/go-xlite/wbx/handler_media"
	handlerproxy "github.com/go-xlite/wbx/handler_proxy"
	handlersse "github.com/go-xlite/wbx/handler_sse"
	handlerws "github.com/go-xlite/wbx/handler_ws"
	"github.com/go-xlite/wbx/webcast"
	"github.com/go-xlite/wbx/weblite"
	"github.com/go-xlite/wbx/webproxy"
	"github.com/go-xlite/wbx/websock"
	"github.com/go-xlite/wbx/webstream"
)

func main() {
	// Create weblite server using provider
	server := weblite.Provider.Servers.New("demo")
	server.SetPort("8080")

	// Initialize the application with embedded files
	appInstance := app.NewApp()

	// Debug: List all embedded files
	debugfs.PrintEmbeddedFiles(appInstance.Content, "[DEBUG] Embedded files:")

	// Create filesystem adapter for embedded files
	embedAdapter := embedfs.NewEmbedFS(&appInstance.Content)
	embedAdapter.SetBasePath("dist") // Set base path inside the embedded FS

	// Create XApp handler for serving HTML applications
	xappHandler := wbx.NewXAppHandler(server)
	xappHandler.SetPathPrefix("xt23")
	xappHandler.SecurityHeaders = true
	xappHandler.VirtualDirSegment = "p"       // Use /p/ for virtual directory
	xappHandler.AuthSkippedPaths = []string{} // No auth for demo

	// Serve from root / (maps to index directory by default)
	// URL: localhost:8080/ or localhost:8080/p/app.js
	// Storage: dist/index/index.html or dist/index/app.js

	// Create webcast server for SSE connections
	sseServer := webcast.NewWebCast()
	server.GetRoutes().ForwardPathPrefixFn("/xt23/sse/", sseServer.OnRequest)

	// Create SSE handler
	sseHandler := handlersse.NewSSEHandler(sseServer)

	// Register SSE stream endpoint
	sseServer.GetRoutes().HandlePathFn("/stream", sseHandler.HandleSSE())

	// Start dummy SSE event streamer
	debugsse.StartDummyStreamer(sseHandler, 3*time.Second)

	// === Webstream (Media Streaming) ===
	// Create filesystem adapter for video data
	videoFsAdapter := osfs.NewOsFsAdapter()
	videoFsAdapter.SetBasePath("../../video_data")

	// Create webstream server
	streamServer := webstream.NewWebStream(videoFsAdapter)

	// Create media handler (thin wrapper)
	mediaHandler := handlermedia.NewMediaHandler(streamServer)

	// === Webproxy (Reverse Proxy) ===
	// Create webproxy server pointing to external service
	proxyServer, err := webproxy.NewWebproxy("https://file-drop.gtn.one:8080/xt21/")
	if err != nil {
		log.Fatalf("Failed to create proxy server: %v", err)
	}

	// Create proxy handler (thin wrapper)
	proxyHandler := handlerproxy.NewProxyHandler(proxyServer)

	// Register SSE routes before static handler
	server.GetRoutes().ForwardPathPrefixFn("/xt23/sse", sseServer.OnRequest)

	// Register media streaming routes before static handler
	mediaHandler.SetPathPrefix("/stream")
	server.GetRoutes().ForwardPathPrefixFn("/xt23/stream", mediaHandler.HandleMedia())

	// Register proxy routes before static handler
	proxyHandler.SetPathPrefix("/proxy")
	server.GetRoutes().ForwardPathPrefixFn("/xt23/proxy", proxyHandler.HandleProxy())

	wsServer := websock.NewWebSock()

	// Create WebSocket handler
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

	xappHandler.ServeStatic("/", embedAdapter)

	// Start the server
	log.Println("Server starting on http://localhost:8080")

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
