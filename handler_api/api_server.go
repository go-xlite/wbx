package weblite

import (
	"embed"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	hl1 "github.com/go-xlite/wbx/helpers"
	"github.com/go-xlite/wbx/webtrail"
)

//go:embed app-dist/*
var content embed.FS

// ApiHandler is optimized for serving API requests, typically returning JSON data
// Features: JSON serialization, CORS support, request validation, error handling
type ApiHandler struct {
	*handler_role.HandlerRole
	Timeout time.Duration
	trail   *webtrail.WebTrail
}

// NewApiHandler creates a new API handler with sensible defaults
func NewApiHandler(server *webtrail.WebTrail) *ApiHandler {
	sr := handler_role.NewHandler()
	sr.CORS.EnableCORS = true
	sr.CORS.CORSOrigins = []string{"*"}

	return &ApiHandler{
		HandlerRole: sr,
		Timeout:     30 * time.Second,
		trail:       server,
	}
}

// HandleAPI registers an API handler with automatic JSON response handling and CORS
func (as *ApiHandler) HandleAPI(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	fullPath := as.PathPrefix.Get() + path
	as.Handler.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		// Apply CORS headers if enabled
		as.HandlerRole.CORS.ApplyCORS(w, r)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	})
}

// HandleJSON registers a handler that returns JSON data
func (as *ApiHandler) HandleJSON(path string, handler func(r *http.Request) (any, error)) {
	as.HandleAPI(path, func(w http.ResponseWriter, r *http.Request) {
		data, err := handler(r)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, data)
	})
}

// HandleGET registers a GET-only endpoint
func (as *ApiHandler) HandleGET(path string, handler func(r *http.Request) (any, error)) {
	as.HandleAPI(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}
		data, err := handler(r)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, data)
	})
}

// HandlePOST registers a POST-only endpoint with JSON body parsing
func (as *ApiHandler) HandlePOST(path string, handler func(r *http.Request, body map[string]any) (any, error)) {
	as.HandleAPI(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, http.StatusBadRequest, "Invalid JSON body")
			return
		}
		defer r.Body.Close()

		data, err := handler(r, body)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		WriteJSON(w, http.StatusOK, data)
	})
}

func (as *ApiHandler) Run() {
	// No-op for now; could be used to initialize resources if needed
	as.trail.GetRoutes().ForwardPathPrefixFn(as.PathPrefix.Suffix("p"), func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			data, _ := content.ReadFile("app-dist" + r.URL.Path)
			hl1.Helpers.WriteJsBytes(w, data)
			return
		}
		hl1.Helpers.WriteNotFound(w)
	})

	as.trail.GetRoutes().ForwardPathPrefixFn(as.PathPrefix.Get(), func(w http.ResponseWriter, r *http.Request) {
		as.trail.OnRequest(w, r)
	})

	as.trail.GetRoutes().ForwardPathFn("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, as.PathPrefix.Suffix("/"), http.StatusMovedPermanently)
	})
}
