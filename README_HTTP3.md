# HTTP/3 Support

This package includes optional HTTP/3 support using compile-time build tags.

## Building with HTTP/3

To enable HTTP/3 support, build with the `http3` tag:

```bash
go build -tags http3
```

## Building without HTTP/3 (default)

By default, HTTP/3 is **not** compiled in:

```bash
go build
```

## Installing HTTP/3 Dependencies

If you want to use HTTP/3, you need to install the required dependency:

```bash
go get github.com/quic-go/quic-go/http3
```

## Usage

The API is the same whether HTTP/3 is compiled in or not. When HTTP/3 is disabled, the functions are stubs that do nothing.

```go
import "github.com/go-xlite/wbx/http3"

// Check if request is using HTTP/3
if http3.IsHTTP3Request(r) {
    // Handle HTTP/3 specific logic
}

// Add Alt-Svc header to advertise HTTP/3
http3.AddHTTP3AltSvcHeader(w, "443")

// Log HTTP/3 status
http3.LogHTTP3Status()

// Get client information
clientInfo := http3.GetClientInfo(r)
```

## Platform Optimization

For optimal HTTP/3 performance, you may need to increase UDP buffer sizes:

### Linux
```bash
sudo sysctl -w net.core.rmem_max=2500000
sudo sysctl -w net.core.wmem_max=2500000
```

Or add to `/etc/sysctl.conf`:
```
net.core.rmem_max=2500000
net.core.wmem_max=2500000
```

### macOS
```bash
sudo sysctl -w net.inet.udp.recvspace=2500000
sudo sysctl -w net.inet.udp.maxdgram=2500000
```

### Windows
Default settings should work fine.

## Version Constants

```go
const (
    HTTPVersion1 = 1  // HTTP/1.1
    HTTPVersion2 = 2  // HTTP/2.0
    HTTPVersion3 = 3  // HTTP/3
)
```
