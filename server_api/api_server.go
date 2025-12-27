package weblite

import (
	"encoding/json"
	"net/http"
	"time"

	serverrole "github.com/go-xlite/wbx/server_role"
	weblite "github.com/go-xlite/wbx/weblite"
)

// ApiServer is optimized for serving API requests, typically returning JSON data
// Features: JSON serialization, CORS support, request validation, error handling
type ApiServer struct {
	*serverrole.ServerRole
	Timeout time.Duration
}

// NewApiServer creates a new API server with sensible defaults
func NewApiServer(wl *weblite.WebLite) *ApiServer {
	sr := &serverrole.ServerRole{
		Server:      wl,
		PathPrefix:  "/api",
		CustomMimes: make(map[string]string),
	}
	sr.CORS.EnableCORS = true
	sr.CORS.CORSOrigins = []string{"*"}

	return &ApiServer{
		ServerRole: sr,
		Timeout:    30 * time.Second,
	}
}

// SetCORS configures CORS settings
func (as *ApiServer) SetCORS(enabled bool, origins ...string) *ApiServer {
	as.ServerRole.CORS.SetCORS(enabled, origins...)
	return as
}

// HandleAPI registers an API handler with automatic JSON response handling and CORS
func (as *ApiServer) HandleAPI(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	fullPath := as.PathPrefix + path
	as.Server.GetRoutes().HandlePathFn(fullPath, func(w http.ResponseWriter, r *http.Request) {
		// Apply CORS headers if enabled
		as.ServerRole.CORS.ApplyCORS(w, r)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	})
}

// HandleJSON registers a handler that returns JSON data
func (as *ApiServer) HandleJSON(path string, handler func(r *http.Request) (any, error)) {
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
func (as *ApiServer) HandleGET(path string, handler func(r *http.Request) (any, error)) {
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
func (as *ApiServer) HandlePOST(path string, handler func(r *http.Request, body map[string]any) (any, error)) {
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

// Start starts the API server
func (as *ApiServer) Start() error {
	return nil
}

// Stop stops the API server
func (as *ApiServer) Stop() error {
	return nil
}

// Response helper functions

// WriteJSON writes a JSON response with the given status code
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// WriteHTML writes an HTML response with the given status code
func WriteHTML(w http.ResponseWriter, statusCode int, html string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(html))
	return err
}

// WriteText writes a plain text response with the given status code
func WriteText(w http.ResponseWriter, statusCode int, text string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	_, err := w.Write([]byte(text))
	return err
}

// WriteError writes an error response as JSON
func WriteError(w http.ResponseWriter, statusCode int, message string) error {
	return WriteJSON(w, statusCode, map[string]any{
		"error":   true,
		"message": message,
		"status":  statusCode,
	})
}

// WriteSuccess writes a success response as JSON
func WriteSuccess(w http.ResponseWriter, data any) error {
	return WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    data,
	})
}

// WriteCreated writes a 201 Created response
func WriteCreated(w http.ResponseWriter, data any) error {
	return WriteJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"data":    data,
	})
}

// WriteNoContent writes a 204 No Content response
func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
