package main

import (
	"fmt"
	"log"
	"time"

	rtx "github.com/go-xlite/rtx"
	wbx "github.com/go-xlite/wbx"
	osfs "github.com/go-xlite/wbx/comm/adapter_fs/os_fs"
	server_data "github.com/go-xlite/wbx/debug/api/server_data"
	debugsse "github.com/go-xlite/wbx/debug/sse"
	client "github.com/go-xlite/wbx/demo/client"
	clientroot "github.com/go-xlite/wbx/demo/client-root"
	handlers "github.com/go-xlite/wbx/handlers"
	webapp "github.com/go-xlite/wbx/roots/webapp"
	servers "github.com/go-xlite/wbx/servers"
	"github.com/go-xlite/wbx/servers/websock"
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
	server.AddPortListener(map[string]string{
		"protocol":            "http",
		"ports":               "8090,8091",
		"addresses":           "0.0.0.0,[::]",
		"https_redirect_port": "8080",
	})
	server.AddPortListener(map[string]string{
		"protocol":        "https",
		"ports":           "8080",
		"addresses":       "0.0.0.0,[::]",
		"ssl_cert_path":   "../../certs/cert",
		"ssl_key_path":    "../../certs/priv",
		"domains_allowed": "localhost,pong.gtn.one",
	})

	// === Server-Sent Events (SSE) ===
	// Create webcast server for SSE connections
	sseServer := servers.NewWebCast()
	// Create SSE handler
	sseHandler := handlers.NewSSEHandler(sseServer)
	sseHandler.SetPathPrefix("/xt23/sse")
	server.GetRoutes().HandlePathPrefixFn(sseHandler.PathPrefix.Get(), sseServer.OnRequest)
	sseHandler.Init()
	// Start dummy SSE event streamer
	debugsse.StartDummyStreamer(sseHandler, 3*time.Second)

	// === Webstream (Media Streaming) ===

	// Create filesystem adapter for video data
	videoFsAdapter := osfs.NewOsFsAdapter()
	videoFsAdapter.SetBasePath("../../video_data")

	// Create webstream server
	streamServer := servers.NewWebStream(videoFsAdapter)
	mediaHandler := handlers.NewMediaHandler(streamServer)
	mediaHandler.SetPathPrefix("/xt23/stream")
	server.GetRoutes().ForwardPathPrefixFn(mediaHandler.PathPrefix.Get(), mediaHandler.HandleMedia())

	// === Webproxy (Reverse Proxy) ===
	// Create webproxy server pointing to external service
	proxyServer, _ := servers.NewWebProxy("https://file-drop.gtn.one:8080/xt21/")
	proxyHandler := handlers.NewProxyHandler(proxyServer)
	proxyHandler.SetPathPrefix("/xt23/proxy")
	server.GetRoutes().HandlePathPrefixFn(proxyHandler.PathPrefix.Get(), proxyHandler.HandleProxy())

	// === WebSocket Handler ===
	wsServer := servers.NewWebSock()
	wsHandler := handlers.NewWsHandler(wsServer, "demo-ws")
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

	// Initialize server data provider
	serversData := server_data.NewServersDataGen()
	serversData.Initialize(80)

	// Setup API routes
	wbtServersApi := servers.NewWebTrail()
	wbtServersApi.GetRoutes().HandlePathFn("/servers/a/list", serversData.HandleListRequest)
	wbtServersApi.GetRoutes().HandlePathFn("/servers/i/{id}/details", serversData.HandleDetailsRequest)
	wbtServersApi.GetRoutes().HandlePathFn("/servers/a/filters", serversData.HandleFiltersRequest)

	apiHandler := wbx.NewApiHandler(wbtServersApi)
	apiHandler.SetPathPrefix("/xt23/trail")
	apiHandler.Run()

	// Initialize the application with embedded files
	clientInstance := client.NewClient()

	// == Sway Handler Setup ===
	sway := servers.NewWebSway()
	sway.FsProvider = clientInstance.Content
	sway.SecurityHeaders = true
	sway.VirtualDirSegment = "p" // Use /p/ for virtual directory

	// Create Sway handler for serving HTML applications
	swayHandler := wbx.NewSwayHandler(sway)
	swayHandler.SetPathPrefix("/xt23")
	swayHandler.AuthSkippedPaths = []string{} // No auth for demo
	swayHandler.Run(server)

	clr := clientroot.NewClientRoot()
	app := webapp.NewWebApp()
	app.Fs = clr.Content
	app.DefaultHome = "/xt23/home"
	server.GetRoutes().HandlePathPrefixFn("/", app.HandleRequest)

	go func() {
		if err := server.Start(); err != nil {
			//log.Fatalf("Server failed to start: %v", err)
			rtx.Rtm.ExitWithErr(1, err)
		}
	}()

	rtx.Rtm.WaitForSIGTERM()
}
