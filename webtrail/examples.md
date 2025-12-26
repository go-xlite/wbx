# WebTrail Usage Guide

## What is WebTrail?

**WebTrail** represents a backend server component that receives requests **after** they've been proxied from a main server. The key concept is that the proxy prefix has already been stripped by the main server before the request reaches WebTrail.

### Architecture Flow

```
Client Request: GET /api/users/123
       ↓
Main Server (WebLite) - strips "/api"
       ↓
WebTrail receives: GET /users/123
```

## Basic Usage

### Example 1: Simple API Backend

```go
package main

import (
    "net/http"
    "github.com/net12labs/vfsql/core-mx/wbx"
)

func main() {
    // Create main server
    mainServer := wbx.NewWebLite("main")
    
    // Create backend WebTrail for API
    apiBackend := wbx.NewWebtrail()
    
    // Register routes WITHOUT the proxy prefix
    // These handlers receive requests with /api already stripped
    apiBackend.Routes.GET("/users", listUsers)
    apiBackend.Routes.GET("/users/{id}", getUser)
    apiBackend.Routes.POST("/users", createUser)
    apiBackend.Routes.DELETE("/users/{id}", deleteUser)
    
    // Proxy all /api/* requests to the backend
    // Main server strips "/api" before forwarding to OnRequest
    mainServer.Routes.HandlePathPrefix("/api/", http.HandlerFunc(apiBackend.OnRequest))
    
    mainServer.Start()
}

func listUsers(w http.ResponseWriter, r *http.Request) {
    // Request path here is "/users", not "/api/users"
    wbx.WriteJSON(w, 200, map[string]string{"status": "listing users"})
}

func getUser(w http.ResponseWriter, r *http.Request) {
    // Extract path variable from gorilla/mux
    vars := mux.Vars(r)
    userID := vars["id"]
    
    wbx.WriteJSON(w, 200, map[string]string{
        "id": userID,
        "name": "John Doe",
    })
}
```

### Example 2: Multiple Backend Services

```go
func main() {
    mainServer := wbx.NewWebLite("main")
    
    // Auth service backend
    authService := wbx.NewWebtrailWithBase("/auth")
    authService.Routes.POST("/login", handleLogin)
    authService.Routes.POST("/logout", handleLogout)
    authService.Routes.GET("/verify", handleVerify)
    
    // User service backend
    userService := wbx.NewWebtrailWithBase("/users")
    userService.Routes.GET("/", listUsers)
    userService.Routes.GET("/{id}", getUser)
    userService.Routes.POST("/", createUser)
    
    // Orders service backend
    orderService := wbx.NewWebtrailWithBase("/orders")
    orderService.Routes.GET("/", listOrders)
    orderService.Routes.GET("/{id}", getOrder)
    
    // Register each service with the main server
    mainServer.Routes.HandlePathPrefix("/auth/", http.HandlerFunc(authService.OnRequest))
    mainServer.Routes.HandlePathPrefix("/users/", http.HandlerFunc(userService.OnRequest))
    mainServer.Routes.HandlePathPrefix("/orders/", http.HandlerFunc(orderService.OnRequest))
    
    mainServer.Start()
}
```

### Example 3: Static Files in Backend

```go
func setupStaticBackend() *wbx.WebTrail {
    backend := wbx.NewWebtrail()
    
    // Serve static files from /assets/*
    // Main server will strip the proxy prefix before this
    backend.Routes.HandlePathPrefix("/assets/", 
        http.FileServer(http.Dir("./public")))
    
    // API routes in the same backend
    backend.Routes.GET("/status", func(w http.ResponseWriter, r *http.Request) {
        wbx.WriteJSON(w, 200, map[string]string{"status": "ok"})
    })
    
    return backend
}

func main() {
    mainServer := wbx.NewWebLite("main")
    cdn := setupStaticBackend()
    
    // Proxy /cdn/* to backend
    // Request to /cdn/assets/style.css becomes /assets/style.css in backend
    mainServer.Routes.HandlePathPrefix("/cdn/", http.HandlerFunc(cdn.OnRequest))
    
    mainServer.Start()
}
```

### Example 4: Path Helpers with PathBase

```go
func setupApiWithBase() {
    // PathBase is for documentation/clarity, NOT used in actual routing
    api := wbx.NewWebtrailWithBase("/api/v1")
    
    // Register routes as normal (without prefix)
    api.Routes.GET("/users", listUsers)
    api.Routes.POST("/users", createUser)
    
    // Use MakePath for logging or documentation
    fmt.Println("Full path would be:", api.MakePath("/users"))
    // Output: Full path would be: /api/v1/users
    
    // But actual routing is still on /users
}
```

## Advanced Features

### Custom 404 Handler

```go
backend := wbx.NewWebtrail()

backend.SetNotFoundHandler(func(w http.ResponseWriter, r *http.Request) {
    wbx.WriteJSON(w, 404, map[string]string{
        "error": "endpoint not found",
        "path": r.URL.Path,
    })
})
```

### Route Introspection

```go
backend := wbx.NewWebtrail()
backend.Routes.GET("/users", listUsers)
backend.Routes.POST("/users", createUser)

// Get all registered routes
routes := backend.Routes.GetRoutes()
for _, route := range routes {
    fmt.Printf("Path: %s, Methods: %s\n", route["path"], route["methods"])
}
```

### Method-Specific Routing

```go
backend := wbx.NewWebtrail()

// Use method-specific helpers
backend.Routes.GET("/resource", handleGet)
backend.Routes.POST("/resource", handlePost)
backend.Routes.PUT("/resource", handlePut)
backend.Routes.PATCH("/resource", handlePatch)
backend.Routes.DELETE("/resource", handleDelete)
backend.Routes.OPTIONS("/resource", handleOptions)
backend.Routes.HEAD("/resource", handleHead)

// Handle ANY HTTP method
backend.Routes.ANY("/webhook", handleWebhook)

// Or use HandleMethod directly for custom methods
backend.Routes.HandleMethod("CUSTOM", "/resource", handleCustom)
```

### Using ANY for Webhooks and CORS

```go
backend := wbx.NewWebtrail()

// Webhook endpoint that accepts any method
backend.Routes.ANY("/webhook", func(w http.ResponseWriter, r *http.Request) {
    // Handle any HTTP method: GET, POST, PUT, DELETE, etc.
    wbx.WriteJSON(w, 200, map[string]string{
        "method": r.Method,
        "path":   r.URL.Path,
    })
})

// CORS preflight + actual endpoint
backend.Routes.OPTIONS("/api/users", handleCORS)
backend.Routes.ANY("/api/users", handleUsersAnyMethod)

// Catch-all health check
backend.Routes.ANY("/health", func(w http.ResponseWriter, r *http.Request) {
    wbx.WriteJSON(w, 200, map[string]string{"status": "healthy"})
})
```

## Best Practices

### 1. **Clear Service Boundaries**
Organize backend services by domain:
- `/auth/*` → authService WebTrail
- `/users/*` → userService WebTrail  
- `/orders/*` → orderService WebTrail

### 2. **No Prefix in Backend Routes**
❌ Wrong:
```go
backend.Routes.GET("/api/users", handler) // Don't include proxy prefix!
```

✅ Correct:
```go
backend.Routes.GET("/users", handler) // Prefix already stripped by main server
```

### 3. **Use PathBase for Documentation**
```go
// PathBase helps with documentation and clarity
api := wbx.NewWebtrailWithBase("/api/v1")

// But routes are still registered without it
api.Routes.GET("/users", handler)

// Use MakePath for logging/debugging
log.Printf("Registering endpoint: %s", api.MakePath("/users"))
// Output: Registering endpoint: /api/v1/users
```

### 4. **Middleware per Service**
```go
func withAuth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Check authentication
        token := r.Header.Get("Authorization")
        if token == "" {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next(w, r)
    }
}

// Apply middleware to specific routes
backend.Routes.GET("/public", publicHandler)
backend.Routes.GET("/private", withAuth(privateHandler))
```

## Common Patterns

### REST API Backend

```go
func setupRESTBackend() *wbx.WebTrail {
    api := wbx.NewWebtrail()
    
    // Collection endpoints
    api.Routes.GET("/items", listItems)
    api.Routes.POST("/items", createItem)
    
    // Resource endpoints
    api.Routes.GET("/items/{id}", getItem)
    api.Routes.PUT("/items/{id}", updateItem)
    api.Routes.PATCH("/items/{id}", patchItem)
    api.Routes.DELETE("/items/{id}", deleteItem)
    
    // Nested resources
    api.Routes.GET("/items/{id}/comments", getItemComments)
    api.Routes.POST("/items/{id}/comments", createItemComment)
    
    return api
}
```

### Versioned API Backend

```go
func setupVersionedAPI() {
    mainServer := wbx.NewWebLite("main")
    
    // V1 API
    v1 := wbx.NewWebtrailWithBase("/api/v1")
    v1.Routes.GET("/users", listUsersV1)
    mainServer.Routes.HandlePathPrefix("/api/v1/", http.HandlerFunc(v1.OnRequest))
    
    // V2 API
    v2 := wbx.NewWebtrailWithBase("/api/v2")
    v2.Routes.GET("/users", listUsersV2)
    mainServer.Routes.HandlePathPrefix("/api/v2/", http.HandlerFunc(v2.OnRequest))
    
    mainServer.Start()
}
```

## Summary

**WebTrail** is designed for backend services that receive pre-processed requests from a main proxy server:

1. ✅ Routes registered WITHOUT proxy prefix
2. ✅ Main server strips prefix before forwarding
3. ✅ Full gorilla/mux routing capabilities (patterns, methods, etc.)
4. ✅ Lightweight and embeddable
5. ✅ Multiple backends can coexist on one main server
