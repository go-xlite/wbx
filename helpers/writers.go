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

func (h *XHelpers) WriteHTMLfromText(w http.ResponseWriter, status int, data string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write([]byte(data))
}

func (h *XHelpers) WriteHTMLfromBytes(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write(data)
}
