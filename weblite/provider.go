package weblite

type WebLiteProvider struct {
	Servers *wlServers
}

func NewWebLiteProvider() *WebLiteProvider {
	ws := &WebLiteProvider{}
	ws.Servers = &wlServers{
		wlp: ws,
	}
	return ws
}

var Provider = NewWebLiteProvider()

type wlServers struct {
	wlp   *WebLiteProvider
	Items map[string]*WebLite
}

func (wls *wlServers) New(name string) *WebLite {
	wl := NewWebLite(name)
	if wls.Items == nil {
		wls.Items = make(map[string]*WebLite)
	}
	wls.Items[name] = wl
	return wl
}

func (wls *wlServers) Get(name string) *WebLite {
	return wls.Items[name]
}

func (wls *wlServers) CloseAll() {
	for name, wl := range wls.Items {
		println("Closing WebLite server:", name)
		if err := wl.Close(); err != nil {
			println("Error closing server", name, ":", err.Error())
		}
	}
}

func (wls *wlServers) List() []*WebLite {
	var list []*WebLite
	for _, wl := range wls.Items {
		list = append(list, wl)
	}
	return list
}

func (wls *wlServers) StopAll() error {
	var errors []error
	for name, wl := range wls.Items {
		println("Stopping WebLite server:", name)
		if err := wl.Stop(); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errors[0]
	}
	return nil
}

func (wls *wlServers) Remove(name string) error {
	wl := wls.Items[name]
	if wl == nil {
		return nil
	}
	if wl.IsRunning() {
		if err := wl.Stop(); err != nil {
			return err
		}
	}
	delete(wls.Items, name)
	return nil
}

func (wls *wlServers) Count() int {
	return len(wls.Items)
}
