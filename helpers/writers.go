package helpers

import (
	"encoding/json"
	"net/http"
)

func (h *XHelpers) WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *XHelpers) WriteHTMLText(w http.ResponseWriter, status int, data string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write([]byte(data))
}

func (h *XHelpers) WriteHTMLBytes(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write(data)
}

func (h *XHelpers) WriteJsText(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(data))
}
func (h *XHelpers) WriteJsBytes(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/javascript")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *XHelpers) WriteCssText(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "text/css")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(data))
}

func (h *XHelpers) WriteCssBytes(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "text/css")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
