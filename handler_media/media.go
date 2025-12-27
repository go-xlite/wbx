package handlermedia

import (
	"net/http"
	"strings"
	"time"

	handler_role "github.com/go-xlite/wbx/comm/handler_role"
	"github.com/go-xlite/wbx/webstream"
)

// MediaHandler handles video and audio streaming with range request support
// This is a thin wrapper that delegates to the webstream server
type MediaHandler struct {
	*handler_role.HandlerRole
	webstream *webstream.Webstream
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(ws *webstream.Webstream) *MediaHandler {
	handlerRole := handler_role.NewHandler()
	handlerRole.Handler = ws

	return &MediaHandler{
		HandlerRole: handlerRole,
		webstream:   ws,
	}
}

// SetBufferSize sets the streaming buffer size
func (mh *MediaHandler) SetBufferSize(size int) *MediaHandler {
	mh.webstream.BufferSize = size
	return mh
}

// SetCaching enables or disables caching
func (mh *MediaHandler) SetCaching(enabled bool, duration time.Duration) *MediaHandler {
	mh.webstream.EnableCaching = enabled
	mh.webstream.CacheDuration = duration
	return mh
}

// AddAllowedExtension adds an allowed file extension
func (mh *MediaHandler) AddAllowedExtension(ext string) *MediaHandler {
	mh.webstream.AddAllowedExtension(ext)
	return mh
}

// ServeMedia serves a media file with range request support
// Delegates to the webstream server
func (mh *MediaHandler) ServeMedia(w http.ResponseWriter, r *http.Request, filePath string) {
	mh.webstream.ServeMedia(w, r, filePath)
}

// HandleMedia creates an HTTP handler for serving media
func (mh *MediaHandler) HandleMedia() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract file path from URL
		filePath := strings.TrimPrefix(r.URL.Path, mh.PathPrefix.Get())
		filePath = strings.TrimPrefix(filePath, "/")

		if filePath == "" {
			http.Error(w, "No media file specified", http.StatusBadRequest)
			return
		}

		mh.ServeMedia(w, r, filePath)
	}
}
