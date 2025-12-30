package swayhandler

import (
	"embed"
	"net/http"
	"strings"

	comm "github.com/go-xlite/wbx/comm"
	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	hl1 "github.com/go-xlite/wbx/helpers"
	"github.com/go-xlite/wbx/server/websway"
	"github.com/go-xlite/wbx/weblite"
)

//go:embed app-dist/*
var content embed.FS

// SwayHandler is optimized for serving HTML applications with linked assets
// Features: Template rendering, asset serving, security headers
type SwayHandler struct {
	*handler_role.HandlerRole
	SessionResolver  comm.SessionResolver
	LoginPage        string
	AuthSkippedPaths []string
	sway             *websway.WebSway
}

// NewSwayHandler creates a SwayHandler wrapper around an existing handler instance
func NewSwayHandler(sway *websway.WebSway) *SwayHandler {
	handlerRole := handler_role.NewHandler()

	return &SwayHandler{
		sway:             sway,
		HandlerRole:      handlerRole,
		LoginPage:        "/login",
		AuthSkippedPaths: []string{"/login", "/logout", "/"},
	}
}

func (ws *SwayHandler) Run(wbl *weblite.WebLite) {

	wbl.GetRoutes().ForwardPathPrefixFn(ws.PathPrefix.Suffix("/sway/p"), func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			data, _ := content.ReadFile("app-dist" + r.URL.Path)
			hl1.Helpers.WriteJsBytes(w, data)
			return
		}
		hl1.Helpers.WriteNotFound(w)
	})

	wbl.GetRoutes().ForwardPathPrefixFn(ws.PathPrefix.Suffix("/"), func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index/p/sw.js" {
			ws.sway.ServeServiceWorker("/index/sw.js", ws.PathPrefix.Suffix("/"), w, r)
			return
		}

		if r.URL.Path == "/site.webmanifest" {
			ws.sway.ServeWebManifest("index/site.webmanifest", ws.PathPrefix.GetNoTrailingSlash(), w, r)
			return
		}

		ws.sway.ServeFile(w, r)
	})

	wbl.GetRoutes().HandlePathFn(ws.PathPrefix.Get(), func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/"
		ws.sway.ServeFile(w, r)
	})

}
