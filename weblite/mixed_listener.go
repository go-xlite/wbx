package weblite

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
)

// mixedProtocolListener wraps a listener to handle both HTTP and HTTPS on the same port
type mixedProtocolListener struct {
	net.Listener
	tlsConfig *tls.Config
	httpsPort string
}

// Accept accepts connections and wraps them to detect HTTP vs HTTPS
func (l *mixedProtocolListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &mixedProtocolConn{
		Conn:      conn,
		tlsConfig: l.tlsConfig,
		httpsPort: l.httpsPort,
	}, nil
}

// mixedProtocolConn wraps a connection to detect and handle HTTP vs HTTPS
type mixedProtocolConn struct {
	net.Conn
	tlsConfig *tls.Config
	httpsPort string
	reader    io.Reader
}

// Read inspects the first bytes to determine if it's HTTP or TLS
func (c *mixedProtocolConn) Read(b []byte) (int, error) {
	if c.reader == nil {
		// Peek at the first few bytes to detect protocol
		br := bufio.NewReader(c.Conn)
		peek, err := br.Peek(6)
		if err != nil {
			return 0, err
		}

		// Check if it's a TLS handshake (starts with 0x16 for TLS handshake)
		// or if it looks like HTTP (starts with "GET ", "POST", "PUT ", "HEAD", etc.)
		isTLS := len(peek) > 0 && peek[0] == 0x16
		isHTTP := bytes.HasPrefix(peek, []byte("GET ")) ||
			bytes.HasPrefix(peek, []byte("POST")) ||
			bytes.HasPrefix(peek, []byte("PUT ")) ||
			bytes.HasPrefix(peek, []byte("HEAD")) ||
			bytes.HasPrefix(peek, []byte("DELE")) ||
			bytes.HasPrefix(peek, []byte("PATC")) ||
			bytes.HasPrefix(peek, []byte("OPTI"))

		if isHTTP && !isTLS {
			// It's plain HTTP on HTTPS port - send redirect
			c.handleHTTPRedirect(br)
			return 0, io.EOF
		}

		// It's TLS or unknown, proceed normally
		c.reader = br
	}

	return c.reader.Read(b)
}

// handleHTTPRedirect sends an HTTP redirect response
func (c *mixedProtocolConn) handleHTTPRedirect(br *bufio.Reader) {
	// Read the HTTP request
	req, err := http.ReadRequest(br)
	if err != nil {
		c.Conn.Close()
		return
	}

	// Extract host without port
	host := req.Host
	if h, _, err := net.SplitHostPort(req.Host); err == nil {
		host = h
	}

	// Build HTTPS URL
	var httpsURL string
	if c.httpsPort == "443" || c.httpsPort == "" {
		httpsURL = fmt.Sprintf("https://%s%s", host, req.RequestURI)
	} else {
		httpsURL = fmt.Sprintf("https://%s:%s%s", host, c.httpsPort, req.RequestURI)
	}

	// Send redirect response
	response := fmt.Sprintf("HTTP/1.1 301 Moved Permanently\r\n"+
		"Location: %s\r\n"+
		"Content-Type: text/html; charset=utf-8\r\n"+
		"Content-Length: %d\r\n"+
		"Connection: close\r\n"+
		"\r\n"+
		"<html><body>Redirecting to <a href=\"%s\">%s</a>...</body></html>",
		httpsURL, len(httpsURL)+55, httpsURL, httpsURL)

	c.Conn.Write([]byte(response))
	c.Conn.Close()
}
