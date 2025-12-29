package weblite

import (
	"embed"
	"net/http"
	"strings"

	comm "github.com/go-xlite/wbx/comm"
	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	hl1 "github.com/go-xlite/wbx/helpers"
	"github.com/go-xlite/wbx/websway"
)

//go:embed app-dist/*
var content embed.FS

// XAppHandler is optimized for serving HTML applications with linked assets
// Features: Template rendering, asset serving, security headers
type XAppHandler struct {
	*handler_role.HandlerRole
	SessionResolver  comm.SessionResolver
	LoginPage        string
	AuthSkippedPaths []string
	sway             *websway.WebSway
}

// NewXAppHandler creates a XAppHandler wrapper around an existing handler instance
func NewXAppHandler(sway *websway.WebSway) *XAppHandler {
	handlerRole := handler_role.NewHandler()

	return &XAppHandler{
		sway:             sway,
		HandlerRole:      handlerRole,
		LoginPage:        "/login",
		AuthSkippedPaths: []string{"/login", "/logout", "/"},
	}
}

func (ws *XAppHandler) Run() {

	ws.sway.GetRoutes().ForwardPathPrefixFn(ws.PathPrefix.Suffix("sway/p"), func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			data, _ := content.ReadFile("app-dist" + r.URL.Path)
			hl1.Helpers.WriteJsBytes(w, data)
			return
		}
		hl1.Helpers.WriteNotFound(w)
	})

	ws.sway.GetRoutes().ForwardPathPrefixFn(ws.PathPrefix.Get(), func(w http.ResponseWriter, r *http.Request) {
		ws.sway.ServeFile(w, r)
	})

	ws.sway.GetRoutes().ForwardPathFn("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, ws.PathPrefix.Suffix("/"), http.StatusMovedPermanently)
	})

}
