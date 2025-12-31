package webauth

import (
	"net/http"

	"github.com/go-xlite/wbx/comm"
)

type IWebAuthProvider interface {
	Login(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	RefreshToken(w http.ResponseWriter, r *http.Request)
	RegisterUser(w http.ResponseWriter, r *http.Request)
	GetCurrentUser(w http.ResponseWriter, r *http.Request)
}

type WebAuth struct {
	*comm.ServerCore
	// Note: The base path is NOT used in actual routing, only for helper methods
	PathBase string // Optional base path for convenience (e.g., "/api" for documentation)
	NotFound http.HandlerFunc
	Auth     IWebAuthProvider
}

// NewWebAuth creates a new WebAuth instance with proper routing capabilities
func NewWebAuth() *WebAuth {
	wt := &WebAuth{
		ServerCore: comm.NewServerCore(),
		PathBase:   "",
	}
	wt.NotFound = http.NotFound
	return wt
}

func (wt *WebAuth) OnRequest(w http.ResponseWriter, r *http.Request) {
	wt.Mux.ServeHTTP(w, r)
}

func (wt *WebAuth) Init() {
	wt.Mux.HandleFunc("/login", wt.Auth.Login)
	wt.Mux.HandleFunc("/logout", wt.Auth.Logout)
	wt.Mux.HandleFunc("/refresh", wt.Auth.RefreshToken)
	wt.Mux.HandleFunc("/register", wt.Auth.RegisterUser)
	wt.Mux.HandleFunc("/me", wt.Auth.GetCurrentUser)
}
