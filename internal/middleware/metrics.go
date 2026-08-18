package middleware

import (
	"time"

	"github.com/asnakech/asnakech-servers/internal/platform/metrics"
	"github.com/gin-gonic/gin"
)

// HTTPMetrics records request counts and latency into the given registry.
func HTTPMetrics(reg *metrics.Registry) gin.HandlerFunc {
	if reg == nil {
		reg = metrics.Default
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		reg.ObserveRequest(c.Request.Method, path, c.Writer.Status(), time.Since(start).Seconds())
	}
}
