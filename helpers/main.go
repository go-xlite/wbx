package helpers

import "net/http"

type XHelpers struct{}

func (h *XHelpers) SetContentTypeBasedOnFileExtension(w http.ResponseWriter, path string) {
	if len(path) == 0 {
		return
	}

	// Set content type based on file extension
	switch {
	case len(path) >= 4 && path[len(path)-4:] == ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case len(path) >= 3 && path[len(path)-3:] == ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case len(path) >= 5 && path[len(path)-5:] == ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case len(path) >= 4 && path[len(path)-4:] == ".json":
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case len(path) >= 4 && (path[len(path)-4:] == ".png"):
		w.Header().Set("Content-Type", "image/png")
	case len(path) >= 4 && (path[len(path)-4:] == ".jpg" || path[len(path)-5:] == ".jpeg"):
		w.Header().Set("Content-Type", "image/jpeg")
	case len(path) >= 4 && path[len(path)-4:] == ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	}
}

var Helpers = &XHelpers{}
