package webapp

import (
	"net/http"

	"github.com/go-xlite/wbx/comm"
	hl1 "github.com/go-xlite/wbx/utils"
)

type WebApp struct {
	*comm.ServerCore
	// Note: The base path is NOT used in actual routing, only for helper methods
	PathBase     string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound     http.HandlerFunc
	Fs           comm.IFsAdapter
	OnRequest    func(w http.ResponseWriter, r *http.Request)
	OnFavIcon    func(w http.ResponseWriter, r *http.Request)
	OnNotFound   func(w http.ResponseWriter, r *http.Request)
	OnServerErr  func(w http.ResponseWriter, r *http.Request)
	OnRootAccess func(w http.ResponseWriter, r *http.Request)
	DefaultHome  string
}

// NewWebApp creates a new WebApp instance with proper routing capabilities
func NewWebApp() *WebApp {
	wt := &WebApp{
		ServerCore: comm.NewServerCore(),
		PathBase:   "",
	}
	wt.NotFound = http.NotFound
	return wt
}
func (wt *WebApp) ServeDefaultFavIcon(w http.ResponseWriter, r *http.Request) bool {
	if wt.OnFavIcon != nil {
		wt.OnFavIcon(w, r)
		return true
	}
	if wt.Fs != nil {
		data, err := wt.Fs.ReadFile("favicon.ico")
		if err == nil {
			hl1.Helpers.WriteFavIcon(w, r, data)
			return true
		}
	}
	return false
}
func (wt *WebApp) serveFsHtml(path string, w http.ResponseWriter, r *http.Request) bool {
	if wt.Fs == nil {
		return false
	}
	data, err := wt.Fs.ReadFile(path)
	if err == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
		return true
	}
	return false
}

func (wt *WebApp) Serve404(w http.ResponseWriter, r *http.Request) {
	if wt.OnNotFound != nil {
		wt.OnNotFound(w, r)
		return
	}
	// serve it with a referrer URL as parameter like /404?referrer=/some/missing/page
	if wt.serveFsHtml("404.html", w, r) {
		return
	}
	wt.NotFound(w, r)
}

func (wt *WebApp) Serve500(w http.ResponseWriter, r *http.Request) {
	if wt.OnServerErr != nil {
		wt.OnServerErr(w, r)
		return
	}
	if wt.serveFsHtml("500.html", w, r) {
		return
	}

	http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
}
func (wt *WebApp) HandleRootAccess(w http.ResponseWriter, r *http.Request) {
	if wt.OnRootAccess != nil {
		wt.OnRootAccess(w, r)
		return
	}
	http.Redirect(w, r, wt.DefaultHome, http.StatusFound)
}

func (wt *WebApp) HandleRequest(w http.ResponseWriter, r *http.Request) {

	if wt.DefaultHome != "" && (r.URL.Path == "/" || r.URL.Path == "") {
		wt.HandleRootAccess(w, r)
		return
	}
	if r.URL.Path == "/favicon.ico" && wt.ServeDefaultFavIcon(w, r) {
		return
	}
	if r.URL.Path == "/404" {
		wt.Serve404(w, r)
		return
	}
	wt.Mux.ServeHTTP(w, r)
}
