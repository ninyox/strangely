package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders applies the response hardening that makes sense for a JSON
// API that browsers call directly. The API only ever answers with JSON, so the
// policy can be extremely strict: nothing may be loaded or framed.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.Writer.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("Cross-Origin-Opener-Policy", "same-origin")
		header.Set("Cross-Origin-Resource-Policy", "same-site")
		header.Set("Permissions-Policy", "camera=(self), microphone=(self), geolocation=(), payment=()")
		header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")
		// Every response is per-visitor (session state, match ids), so no
		// shared or browser cache may keep a copy.
		header.Set("Cache-Control", "no-store")
		c.Next()
	}
}

// CORSHeaders is used by the WebSocket upgrade path, which gin-contrib/cors
// does not cover: a WebSocket handshake is a plain GET with an Origin header.
func CORSHeaders(allowedOrigins []string, requestOrigin string) (string, bool) {
	if requestOrigin == "" {
		return "", false
	}
	for _, allowed := range allowedOrigins {
		if allowed == requestOrigin {
			return requestOrigin, true
		}
	}
	return "", false
}
