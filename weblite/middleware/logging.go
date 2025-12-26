package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// Logging creates a middleware that logs HTTP requests
func Logging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(lrw, r)

			// Log request
			duration := time.Since(start)
			log.Printf(
				"%s %s %d %s %s",
				r.Method,
				r.RequestURI,
				lrw.statusCode,
				duration,
				r.RemoteAddr,
			)
		})
	}
}

// LoggingWithFormat creates a custom logging middleware with a format function
func LoggingWithFormat(format func(*http.Request, int, time.Duration) string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(lrw, r)

			duration := time.Since(start)
			log.Println(format(r, lrw.statusCode, duration))
		})
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture the status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

// Recovery creates a middleware that recovers from panics
func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("PANIC: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryWithHandler creates a middleware that recovers from panics with a custom handler
func RecoveryWithHandler(handler func(http.ResponseWriter, *http.Request, any)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("PANIC: %v", err)
					handler(w, r, err)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID generates a unique ID for each request
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := fmt.Sprintf("%d", time.Now().UnixNano())
			r = SetInContext(r, RequestIDKey, requestID)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r)
		})
	}
}
