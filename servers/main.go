package servers

import (
	webcast "github.com/go-xlite/wbx/servers/webcast"
	webproxy "github.com/go-xlite/wbx/servers/webproxy"
	websock "github.com/go-xlite/wbx/servers/websock"
	webstream "github.com/go-xlite/wbx/servers/webstream"
	websway "github.com/go-xlite/wbx/servers/websway"
	webtrail "github.com/go-xlite/wbx/servers/webtrail"
)

type WebCast = webcast.WebCast
type WebProxy = webproxy.WebProxy
type WebSock = websock.WebSock
type WebStream = webstream.WebStream
type WebSway = websway.WebSway
type WebTrail = webtrail.WebTrail

var NewWebCast = webcast.NewWebCast
var NewWebProxy = webproxy.NewWebProxy
var NewWebSock = websock.NewWebSock
var NewWebStream = webstream.NewWebStream
var NewWebSway = websway.NewWebSway
var NewWebTrail = webtrail.NewWebTrail
