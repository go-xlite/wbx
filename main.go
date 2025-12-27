package wbx

import (
	"github.com/go-xlite/wbx/comm"
	hl1 "github.com/go-xlite/wbx/helpers"
	sa "github.com/go-xlite/wbx/server_api"
	sc "github.com/go-xlite/wbx/server_cdn"
	sw "github.com/go-xlite/wbx/server_web"
	wl "github.com/go-xlite/wbx/weblite"
	wr "github.com/go-xlite/wbx/webrock"
	wt "github.com/go-xlite/wbx/webtrail"
)

var WebLites = wl.Provider

// Type aliases for easy access
type WebLite = wl.WebLite
type ApiServer = sa.ApiServer
type CdnServer = sc.CdnServer
type WebServer = sw.WebServer

type WebRock = wr.WebRock

// Constructor functions
var NewWebLite = wl.NewWebLite
var NewApiServer = sa.NewApiServer
var NewCdnServer = sc.NewCdnServer
var NewWebServer = sw.NewWebServer

// Utility functions
var GetMimeType = comm.GetMimeType
var WriteJSON = hl1.Helpers.WriteJSON
var WriteHTML = hl1.Helpers.WriteHTMLfromText
var WriteHTMLBytes = hl1.Helpers.WriteHTMLfromBytes

var NewWebTrail = wt.NewWebtrail
var NewWebTrailWithBase = wt.NewWebtrailWithBase
var NewWebRock = wr.NewWebRock

type WebTrail = wt.WebTrail

type helpers struct {
	*hl1.XHelpers
}

var Hx = &helpers{
	XHelpers: hl1.Helpers,
}
