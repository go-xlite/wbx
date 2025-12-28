package main

import (
	"fmt"
	"io/fs"
	"log"
	"time"

	wbx "github.com/go-xlite/wbx" // Import to include XAppHandler
	embedfs "github.com/go-xlite/wbx/adapter_fs/embed_fs"
	osfs "github.com/go-xlite/wbx/adapter_fs/os_fs"
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
	fmt.Println("[DEBUG] Embedded files:")
	fs.WalkDir(appInstance.Content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fmt.Printf("  - %s\n", path)
		}
		return nil
	})

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
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		counter := 0
		for range ticker.C {
			counter++
			message := fmt.Sprintf("Server event #%d at %s", counter, time.Now().Format("15:04:05"))
			count := sseHandler.Broadcast(message)
			if count > 0 {
				log.Printf("[SSE] Broadcast message to %d clients: %s", count, message)
			}
		}
	}()

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

	// Create websock server for WebSocket connections
	wsServer := websock.NewWebSock()
	server.GetRoutes().ForwardPathPrefixFn("/xt23/ws/", wsServer.OnRequest)

	go wsServer.Run()

	// Register WebSocket routes before static handler
	server.GetRoutes().ForwardPathPrefixFn("/xt23/ws", wsServer.OnRequest)

	// Register SSE routes before static handler
	server.GetRoutes().ForwardPathPrefixFn("/xt23/sse", sseServer.OnRequest)

	// Register media streaming routes before static handler
	mediaHandler.SetPathPrefix("/stream")
	server.GetRoutes().ForwardPathPrefixFn("/xt23/stream", mediaHandler.HandleMedia())

	// Register proxy routes before static handler
	proxyHandler.SetPathPrefix("/proxy")
	server.GetRoutes().ForwardPathPrefixFn("/xt23/proxy", proxyHandler.HandleProxy())

	xappHandler.ServeStatic("/", embedAdapter)

	// Create WebSocket handler
	wsHandler := handlerws.NewWsHandler(wsServer, "demo-ws")
	wsHandler.OnMessage = func(clientID string, userID int64, username string, message []byte) {
		log.Printf("[WebSocket] Message from %s (%s): %s", username, clientID, string(message))
		// Echo message back to client
		wsServer.SendToClient(clientID, []byte(fmt.Sprintf("Echo: %s", string(message))))
	}
	wsHandler.OnConnect = func(clientID string, userID int64, username string) {
		log.Printf("[WebSocket] Client connected: %s (%s)", username, clientID)
	}
	wsHandler.OnDisconnect = func(clientID string, userID int64, username string) {
		log.Printf("[WebSocket] Client disconnected: %s (%s)", username, clientID)
	}
	wsHandler.Run()

	// Start the server
	log.Println("Server starting on http://localhost:8080")
	log.Println("Visit:")
	log.Println("  - http://localhost:8080/xt23/ (serves from index directory)")
	log.Println("  - http://localhost:8080/xt23/home")
	log.Println("  - http://localhost:8080/xt23/ws-test/ (WebSocket test console)")
	log.Println("  - http://localhost:8080/xt23/sse-test/ (SSE test console)")
	log.Println("  - http://localhost:8080/xt23/stream-test/ (Media streaming test console)")
	log.Println("  - http://localhost:8080/xt23/proxy-test/ (Reverse proxy test console)")
	log.Println("  - http://localhost:8080/xt23/p/app.js (asset via virtual directory)")
	log.Println("  - http://localhost:8080/xt23/sse/stream (SSE endpoint)")
	log.Println("  - http://localhost:8080/xt23/stream/sharko_video.mp4 (Video stream endpoint)")
	log.Println("  - http://localhost:8080/xt23/proxy/ (Reverse proxy to https://file-drop.gtn.one:8080/xt21/)")
	log.Println("  - ws://localhost:8080/xt23/ws/connect (WebSocket endpoint)")
	log.Println("  - http://localhost:8080/xt23/ws/manager.js (WebSocket manager script)")

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
