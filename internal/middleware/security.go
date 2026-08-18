package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig controls baseline HTTP security headers for API responses.
type SecurityHeadersConfig struct {
	// HSTS enables Strict-Transport-Security (only when TLS terminates at the edge
	// or the app serves HTTPS directly).
	HSTS bool
}

// SecurityHeaders sets common security headers on every response.
func SecurityHeaders(cfg SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// JSON API — no inline scripts or styles expected.
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		if cfg.HSTS {
			// 1 year, include subdomains when behind a trusted TLS terminator.
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// SkipPaths returns true when path matches any exact or prefix skip entry.
func SkipPaths(path string, skips []string) bool {
	for _, skip := range skips {
		if skip == "" {
			continue
		}
		if path == skip || strings.HasPrefix(path, skip) {
			return true
		}
	}
	return false
}
