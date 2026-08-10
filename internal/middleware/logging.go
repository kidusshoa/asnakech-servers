package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ZerologRequest logs each HTTP request with method, path, status, latency,
// and request_id using structured fields.
func ZerologRequest(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		if raw != "" {
			path = path + "?" + raw
		}

		evt := logger.Info()
		if status >= 500 {
			evt = logger.Error()
		} else if status >= 400 {
			evt = logger.Warn()
		}

		evt.
			Str("request_id", c.GetString("request_id")).
			Str("method", c.Request.Method).
			Str("path", path).
			Int("status", status).
			Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Int("bytes", c.Writer.Size()).
			Msg("request")
	}
}
