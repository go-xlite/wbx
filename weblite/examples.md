# WebLite Server Examples

## Basic WebLite Usage

```go
// Create a basic web server
wl := weblite.NewWebLite("my-server")
wl.SetPort("8080")
wl.SetBindAddr("0.0.0.0")

// Add routes
wl.Routes.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello World"))
})

// Start server (blocking)
wl.Start()

// Or start in background
wl.StartBackground()
```

## API Server

Optimized for JSON API endpoints with CORS support and automatic error handling.

```go
// Create API server
api := weblite.NewApiServer("my-api")
api.SetPort("8080")
api.SetPathPrefix("/api/v1")
api.SetCORS(true, "https://example.com", "https://app.example.com")

// Simple GET endpoint
api.HandleGET("/users", func(r *http.Request) (any, error) {
    users := []map[string]string{
        {"id": "1", "name": "Alice"},
        {"id": "2", "name": "Bob"},
    }
    return users, nil
})

// POST endpoint with JSON body
api.HandlePOST("/users", func(r *http.Request, body map[string]any) (any, error) {
    name := body["name"].(string)
    // ... create user logic
    return map[string]any{
        "id": "123",
        "name": name,
        "created": true,
    }, nil
})

// Custom API handler with full control
api.HandleAPI("/custom", func(w http.ResponseWriter, r *http.Request) {
    weblite.WriteSuccess(w, map[string]any{
        "message": "Custom response",
    })
})

api.StartBackground()
```

## CDN Server

Optimized for serving static assets with proper MIME types and caching.

```go
// Create CDN server
cdn := weblite.NewCdnServer("my-cdn")
cdn.SetPort("8080")
cdn.SetPathPrefix("/cdn")
cdn.SetCaching(24*time.Hour, true) // 24h cache, browser caching enabled

// Serve entire directory
cdn.ServeDirectory("/assets", "./public/assets")

// Serve single file
cdn.ServeFile("/logo.png", "./assets/logo.png")

// Serve from embedded filesystem
//go:embed assets/*
var embedFS embed.FS
cdn.ServeEmbedded("/static", &embedFS, "assets")

// Serve single embedded file
cdn.ServeEmbeddedFile("/favicon.ico", &embedFS, "assets/favicon.ico")

// Serve raw bytes
imageData := []byte{...}
cdn.ServeBytes("/dynamic.png", imageData, "image/png")

cdn.StartBackground()
```

## Web Server

Optimized for serving HTML applications with templates and SPA support.

```go
// Create web server
web := weblite.NewWebServer("my-web")
web.SetPort("8080")
web.SetStaticDir("./public")

// Serve static files (JS, CSS, images)
web.ServeStatic("/assets/", "./public/assets")

// Serve HTML page
web.ServeHTML("/", "./public/index.html")

// Single Page Application mode
web.EnableSPAMode()
web.ServeSPA("./public/index.html")

// Or with embedded filesystem
//go:embed public/*
var publicFS embed.FS
web.ServeSPAEmbedded(&publicFS, "public/index.html")

// Template rendering
web.SetTemplateDir("./templates")
web.RenderTemplate("/profile", "profile.html", func(r *http.Request) any {
    return map[string]any{
        "username": "john",
        "email": "john@example.com",
    }
})

// Custom page handler
web.HandlePage("/about", func(w http.ResponseWriter, r *http.Request) {
    weblite.WriteHTMLString(w, "<h1>About Us</h1>")
})

// Redirects
web.Redirect("/old-path", "/new-path", false)
web.Redirect("/moved", "/new-location", true) // permanent

web.StartBackground()
```

## Performance Optimization

### Stats Tracking

```go
wl := weblite.NewWebLite("my-server")

// Get statistics
stats := wl.GetStats()
fmt.Printf("Total requests: %d\n", stats.TotalRequests)
fmt.Printf("Active requests: %d\n", stats.ActiveRequests)
fmt.Printf("Requests per second: %.2f\n", stats.RequestsPerSec)

// Disable detailed tracking for maximum performance
wl.DisableDetailedStats()

// Re-enable if needed
wl.EnableDetailedStats()

// Reset stats
wl.ResetStats()
```

### Fast Stats Middleware

```go
// For high-throughput endpoints, use fast stats tracking
wl.Routes.HandleFuncWithStatsFast("/api/metrics", handler)

// vs regular stats (tracks path and status code details)
wl.Routes.HandleFuncWithStats("/api/users", handler)
```

## SSL/TLS Configuration

```go
wl := weblite.NewWebLite("secure-server")

// From files
wl.SetSSL("./cert.pem", "./key.pem")

// From raw text/bytes
certPEM := `-----BEGIN CERTIFICATE-----...`
keyPEM := `-----BEGIN PRIVATE KEY-----...`
wl.SetSSLFromText(certPEM, keyPEM)

// Or from bytes
wl.SetSSLFromData([]byte(certPEM), []byte(keyPEM))

wl.Start()
```

## Multiple Bind Addresses

```go
wl := weblite.NewWebLite("multi-bind")
wl.SetPort("8080")

// Bind to IPv4 and IPv6
wl.SetBindAddr("0.0.0.0", "::")

// Or specific interfaces
wl.SetBindAddr("127.0.0.1", "192.168.1.100")

wl.Start()
```

## Server Management

```go
wl := weblite.NewWebLite("managed-server")

// Start in background
if err := wl.StartBackground(); err != nil {
    log.Fatal(err)
}

// Check if running
if wl.IsRunning() {
    fmt.Println("Server is running")
}

// Get addresses
addrs := wl.GetAddr()
fmt.Println("Listening on:", addrs)

// Graceful shutdown (with timeout)
wl.Stop()

// Or immediate close
wl.Close()
```

## Response Helpers

### API Responses

```go
// JSON response
weblite.WriteJSON(w, 200, data)

// Success response
weblite.WriteSuccess(w, data)

// Created response (201)
weblite.WriteCreated(w, data)

// Error response
weblite.WriteError(w, 400, "Invalid input")

// No content (204)
weblite.WriteNoContent(w)
```

### Web Responses

```go
// HTML response
weblite.WriteHTML(w, 200, "<h1>Hello</h1>")

// Text response
weblite.WriteText(w, 200, "Plain text")

// Template response
weblite.WriteHTMLTemplate(w, tmpl, "page.html", data)

// Redirect
weblite.WriteRedirect(w, r, "/new-path", false)
```

### CDN Responses

```go
// Binary file
weblite.WriteBinary(w, data, "image.png")

// Force download
weblite.WriteDownload(w, data, "document.pdf")

// Stream file
weblite.StreamFile(w, "/path/to/large-file.mp4")
```

## MIME Types

```go
// Get MIME type by extension
mimeType := weblite.GetMimeType(".png")  // "image/png"
mimeType := weblite.GetMimeType(".json") // "application/json"

// Custom MIME types in CDN server
cdn := weblite.NewCdnServer("cdn")
cdn.AddCustomMime(".custom", "application/x-custom")
```

## Complete Example

```go
package main

import (
    "log"
    "github.com/net12labs/vfsql/core-mx/wbx/weblite"
)

func main() {
    // API Server for REST endpoints
    api := weblite.NewApiServer("api")
    api.SetPort("8080")
    api.HandleGET("/health", func(r *http.Request) (any, error) {
        return map[string]string{"status": "ok"}, nil
    })
    
    // CDN Server for static assets
    cdn := weblite.NewCdnServer("cdn")
    cdn.SetPort("8081")
    cdn.ServeDirectory("/assets", "./public/assets")
    
    // Web Server for HTML pages
    web := weblite.NewWebServer("web")
    web.SetPort("8082")
    web.EnableSPAMode()
    web.ServeSPA("./public/index.html")
    web.ServeStatic("/static/", "./public/static")
    
    // Start all servers in background
    if err := api.StartBackground(); err != nil {
        log.Fatal(err)
    }
    if err := cdn.StartBackground(); err != nil {
        log.Fatal(err)
    }
    if err := web.StartBackground(); err != nil {
        log.Fatal(err)
    }
    
    // Keep running
    select {}
}
```
