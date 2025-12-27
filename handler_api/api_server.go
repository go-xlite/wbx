package weblite

import (
	"encoding/json"
	"net/http"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
)

// ApiHandler is optimized for serving API requests, typically returning JSON data
// Features: JSON serialization, CORS support, request validation, error handling
type ApiHandler struct {
	*handler_role.HandlerRole
	Timeout time.Duration
}

// NewApiHandler creates a new API handler with sensible defaults
func NewApiHandler(wl handler_role.IHandler) *ApiHandler {
	sr := &handler_role.HandlerRole{
		Handler:     wl,
		PathPrefix:  "/api",
		CustomMimes: make(map[string]string),
	}
	sr.CORS.EnableCORS = true
	sr.CORS.CORSOrigins = []string{"*"}

	return &ApiHandler{
		HandlerRole: sr,
		Timeout:     30 * time.Second,
	}
}

// HandleAPI registers an API handler with automatic JSON response handling and CORS
func (as *ApiHandler) HandleAPI(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	fullPath := as.PathPrefix + path
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

// Response helper functions

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response as JSON
func WriteError(w http.ResponseWriter, statusCode int, message string) error {
	return WriteJSON(w, statusCode, map[string]any{
		"error":   true,
		"message": message,
		"status":  statusCode,
	})
}
