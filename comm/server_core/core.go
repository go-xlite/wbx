package servercore

import (
	"github.com/go-xlite/wbx/comm/routes"
	"github.com/gorilla/mux"
)

type ServerCore struct {
	Mux    *mux.Router
	Routes *routes.Routes
	// Core server fields can be added here
}

func (sc *ServerCore) GetMux() *mux.Router {
	return sc.Mux
}
func (sc *ServerCore) GetRoutes() *routes.Routes {
	return sc.Routes
}

func NewServerCore() *ServerCore {
	sc := &ServerCore{}
	sc.Mux = mux.NewRouter()
	sc.Routes = routes.NewRoutes(sc.Mux)
	return sc
}
