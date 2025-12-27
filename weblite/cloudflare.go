package weblite

import (
	"context"
	"net"
	"syscall"
)

// createListenerControl creates a listener control function for CloudFlare optimizations
// Sets TCP_MAXSEG to 1220 to avoid PMTU issues with Cloudflare's IPv6 tunnels
func createListenerControl() func(network, address string, c syscall.RawConn) error {
	return func(network, address string, c syscall.RawConn) error {
		var sockOptErr error
		err := c.Control(func(fd uintptr) {
			// Set TCP_MAXSEG to 1220 to avoid PMTU issues with Cloudflare's IPv6 tunnels
			// This is especially important for IPv6 connections through Cloudflare
			sockOptErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_MAXSEG, 1220)
		})
		if err != nil {
			return err
		}
		return sockOptErr
	}
}

// CreateCloudFlareListener creates a listener with CloudFlare optimizations
func (wl *WebLite) CreateCloudFlareListener(network, addr string) (net.Listener, error) {
	lc := &net.ListenConfig{
		Control: createListenerControl(),
	}
	return lc.Listen(context.Background(), network, addr)
}
