package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (h *XHelpers) WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *XHelpers) WriteHTMLText(w http.ResponseWriter, status int, data string) {
	h.WriteHTMLBytes(w, status, []byte(data))
}

func (h *XHelpers) WriteHTMLBytes(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write(data)
}

func (h *XHelpers) WriteJsText(w http.ResponseWriter, data string) {
	h.WriteJsBytes(w, []byte(data))
}
func (h *XHelpers) WriteJsBytes(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *XHelpers) WriteCssText(w http.ResponseWriter, data string) {
	h.WriteCssBytes(w, []byte(data))
}

func (h *XHelpers) WriteCssBytes(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", Mime.Type.Css)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *XHelpers) WriteNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 - Not Found"))
}

func (h *XHelpers) WriteInternalError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("500 - Internal Server Error: " + err.Error()))
}

func (h *XHelpers) WriteWebManifestBytes(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/manifest+json")
	// Security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(86400*7))) // 7 days
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
func (h *XHelpers) WriteWebManifestText(w http.ResponseWriter, data string) {
	h.WriteWebManifestBytes(w, []byte(data))
}

func (h *XHelpers) WriteRobotsTxt(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "text/plain")
	// Security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(3600))) // 1 hour
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(data))
}

func (h *XHelpers) WriteSitemapXML(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/xml")
	// Security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(3600))) // 1 hour
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(data))
}

func (h *XHelpers) WriteTextPlain(w http.ResponseWriter, status int, data string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(data))
}

func (h *XHelpers) WriteTextPlainBytes(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write(data)
}
func (h *XHelpers) WriteFavIcon(w http.ResponseWriter, r *http.Request, data []byte) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.Write(data)
}
