package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig controls cross-origin behavior.
type CORSConfig struct {
	// AllowedOrigins is a list of origins. Use "*" only for open public APIs
	// without credentials. Never combine "*" with AllowCredentials=true.
	AllowedOrigins []string
	AllowCredentials bool
}

// CORS returns middleware that enforces the given CORS policy.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	allowAll := false
	for _, origin := range cfg.AllowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	// Browsers reject Access-Control-Allow-Origin: * with credentials.
	if allowAll && cfg.AllowCredentials {
		cfg.AllowCredentials = false
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		switch {
		case allowAll && !cfg.AllowCredentials:
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		case origin != "":
			if _, ok := allowed[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Add("Vary", "Origin")
			}
		}

		if cfg.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, X-Request-ID, Idempotency-Key, Accept-Language")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
