package wbx

import (
	"github.com/go-xlite/wbx/comm"
	sa "github.com/go-xlite/wbx/handler_api"
	sc "github.com/go-xlite/wbx/handler_cdn"
	rth "github.com/go-xlite/wbx/handler_root"
	spa "github.com/go-xlite/wbx/handler_spa"
	sw "github.com/go-xlite/wbx/handler_web"
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
type WebHandler = sw.WebHandler
type SpaHandler = spa.SPAHandler
type RootHandler = rth.RootHandler

// Constructor functions
var NewApiHandler = sa.NewApiHandler
var NewCdnHandler = sc.NewCdnHandler
var NewWebHandler = sw.NewWebHandler
var NewSpaHandler = spa.NewSPAHandler
var NewRootHandler = rth.NewRootHandler

// Utility functions
var GetMimeType = comm.GetMimeType
var WriteJSON = hl1.Helpers.WriteJSON
var WriteHTML = hl1.Helpers.WriteHTMLfromText
var WriteHTMLBytes = hl1.Helpers.WriteHTMLfromBytes

type helpers struct {
	*hl1.XHelpers
}

var Hx = &helpers{
	XHelpers: hl1.Helpers,
}
