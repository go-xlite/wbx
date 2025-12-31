package servers

import (
	webcast "github.com/go-xlite/wbx/services/webcast"
	webproxy "github.com/go-xlite/wbx/services/webproxy"
	websock "github.com/go-xlite/wbx/services/websock"
	webstream "github.com/go-xlite/wbx/services/webstream"
	websway "github.com/go-xlite/wbx/services/websway"
	webtrail "github.com/go-xlite/wbx/services/webtrail"
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
