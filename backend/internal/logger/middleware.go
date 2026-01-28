package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// RequestIDKey is the context key for request ID
type RequestIDKey string

const RequestIDContextKey RequestIDKey = "request_id"

// generateRequestID generates a random request ID
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails (shouldn't happen)
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405")))
	}
	return hex.EncodeToString(b)
}

// RequestIDMiddleware adds a request ID to each request and logs request details
func RequestIDMiddleware(logger *Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID
		requestID := generateRequestID()
		
		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)
		r = r.WithContext(ctx)
		
		// Log request start
		start := time.Now()
		logger.WithContext(ctx).Info("request_started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// Call next handler
		next.ServeHTTP(wrapped, r)
		
		// Log request completion
		duration := time.Since(start)
		logger.WithContext(ctx).Info("request_completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
