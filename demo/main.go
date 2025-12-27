package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"

	wbx "github.com/go-xlite/wbx" // Import to include XAppHandler
	embedfs "github.com/go-xlite/wbx/adapter_fs/embed_fs"
	"github.com/go-xlite/wbx/demo/app"
	handlerws "github.com/go-xlite/wbx/handler_ws"
	"github.com/go-xlite/wbx/weblite"
	"github.com/go-xlite/wbx/websock"
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

	// Create websock server for WebSocket connections
	wsServer := websock.NewWebsock()
	server.GetRoutes().HandlePathPrefixFn("/xt23/ws/", func(w http.ResponseWriter, r *http.Request) {
		// Strip the /xt23/ws prefix since weblite mode 0 doesn't strip it automatically
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/xt23/ws")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		log.Printf("[DEMO] Forwarding to websock: %s", r.URL.Path)
		wsServer.OnRequest(w, r)
	})

	go wsServer.Run()

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
	log.Println("  - http://localhost:8080/xt23/p/app.js (asset via virtual directory)")
	log.Println("  - ws://localhost:8080/xt23/ws/connect (WebSocket endpoint)")
	log.Println("  - http://localhost:8080/xt23/ws/manager.js (WebSocket manager script)")

	if err := server.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
