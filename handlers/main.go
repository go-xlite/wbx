package handlers

import (
	handlermedia "github.com/go-xlite/wbx/handlers/handler_media"
	handlerproxy "github.com/go-xlite/wbx/handlers/handler_proxy"
	handlersse "github.com/go-xlite/wbx/handlers/handler_sse"
	handlerws "github.com/go-xlite/wbx/handlers/handler_ws"
)

var NewMediaHandler = handlermedia.NewMediaHandler
var NewProxyHandler = handlerproxy.NewProxyHandler
var NewSSEHandler = handlersse.NewSSEHandler
var NewWsHandler = handlerws.NewWsHandler

type MediaHandler = handlermedia.MediaHandler
type ProxyHandler = handlerproxy.ProxyHandler
type SSEHandler = handlersse.SSEHandler
type WsHandler = handlerws.WsHandler
