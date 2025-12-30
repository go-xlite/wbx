package wbx

import (
	sa "github.com/go-xlite/wbx/handler_api"
	sc "github.com/go-xlite/wbx/handler_cdn"
	media "github.com/go-xlite/wbx/handler_media"
	pxy "github.com/go-xlite/wbx/handler_proxy"
	rth "github.com/go-xlite/wbx/handler_root"
	sse "github.com/go-xlite/wbx/handler_sse"
	sway "github.com/go-xlite/wbx/handler_sway"
	ws "github.com/go-xlite/wbx/handler_ws"
	hl1 "github.com/go-xlite/wbx/helpers"
	wc "github.com/go-xlite/wbx/server/webcast"
	"github.com/go-xlite/wbx/server/webproxy"
	wss "github.com/go-xlite/wbx/server/websock"
	"github.com/go-xlite/wbx/server/webstream"
	wt "github.com/go-xlite/wbx/server/webtrail"
	wl "github.com/go-xlite/wbx/weblite"
)

var WebLites = wl.Provider

type WebTrail = wt.WebTrail
type WebLite = wl.WebLite
type WebCast = wc.WebCast
type WebSock = wss.WebSock
type WebStream = webstream.WebStream
type WebProxy = webproxy.Webproxy

var NewWebLite = wl.NewWebLite
var NewWebTrail = wt.NewWebtrail
var NewWebCast = wc.NewWebCast
var NewWebSock = wss.NewWebSock
var NewWebStream = webstream.NewWebStream
var NewWebProxy = webproxy.NewWebproxy

// Type aliases for easy access
type ApiHandler = sa.ApiHandler
type CdnHandler = sc.CdnHandler
type SwayHandler = sway.SwayHandler
type RootHandler = rth.RootHandler
type SSEHandler = sse.SSEHandler
type WsHandler = ws.WsHandler
type ProxyHandler = pxy.ProxyHandler
type MediaHandler = media.MediaHandler

// Constructor functions
var NewApiHandler = sa.NewApiHandler
var NewCdnHandler = sc.NewCdnHandler
var NewSwayHandler = sway.NewSwayHandler
var NewRootHandler = rth.NewRootHandler
var NewSSEHandler = sse.NewSSEHandler
var NewWsHandler = ws.NewWsHandler
var NewProxyHandler = pxy.NewProxyHandler
var NewMediaHandler = media.NewMediaHandler

// Utility functions
var WriteJSON = hl1.Helpers.WriteJSON
var WriteHTMLText = hl1.Helpers.WriteHTMLText
var WriteHTMLBytes = hl1.Helpers.WriteHTMLBytes
var WriteJsBytes = hl1.Helpers.WriteJsBytes
var WriteJsText = hl1.Helpers.WriteJsText
var WriteCssBytes = hl1.Helpers.WriteCssBytes
var WriteCssText = hl1.Helpers.WriteCssText

type helpers struct {
	*hl1.XHelpers
}

var Hx = &helpers{
	XHelpers: hl1.Helpers,
}
