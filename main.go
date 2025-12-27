package wbx

import (
	sa "github.com/go-xlite/wbx/handler_api"
	sc "github.com/go-xlite/wbx/handler_cdn"
	media "github.com/go-xlite/wbx/handler_media"
	pxy "github.com/go-xlite/wbx/handler_proxy"
	rth "github.com/go-xlite/wbx/handler_root"
	spa "github.com/go-xlite/wbx/handler_spa"
	sse "github.com/go-xlite/wbx/handler_sse"
	ws "github.com/go-xlite/wbx/handler_ws"
	xapp "github.com/go-xlite/wbx/handler_xapp"
	hl1 "github.com/go-xlite/wbx/helpers"
	wl "github.com/go-xlite/wbx/weblite"
	wt "github.com/go-xlite/wbx/webtrail"
)

var WebLites = wl.Provider

type WebTrail = wt.WebTrail
type WebLite = wl.WebLite

var NewWebLite = wl.NewWebLite
var NewWebTrail = wt.NewWebtrail

// Type aliases for easy access
type ApiHandler = sa.ApiHandler
type CdnHandler = sc.CdnHandler
type XAppHandler = xapp.XAppHandler
type SpaHandler = spa.SPAHandler
type RootHandler = rth.RootHandler
type SSEHandler = sse.SSEHandler
type WsHandler = ws.WsHandler
type ProxyHandler = pxy.ProxyHandler
type MediaHandler = media.MediaHandler

// Constructor functions
var NewApiHandler = sa.NewApiHandler
var NewCdnHandler = sc.NewCdnHandler
var NewXAppHandler = xapp.NewXAppHandler
var NewSpaHandler = spa.NewSPAHandler
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
