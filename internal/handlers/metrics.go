package handlers

import (
	"net/http"

	"github.com/asnakech/asnakech-servers/internal/platform/metrics"
	"github.com/gin-gonic/gin"
)

// MetricsHandler exposes Prometheus text metrics.
type MetricsHandler struct {
	reg *metrics.Registry
}

func NewMetricsHandler(reg *metrics.Registry) *MetricsHandler {
	if reg == nil {
		reg = metrics.Default
	}
	return &MetricsHandler{reg: reg}
}

// Prometheus godoc
// @Summary      Prometheus metrics
// @Description  Prometheus text exposition format. Restrict network access in production.
// @Tags         ops
// @Produce      plain
// @Success      200 {string} string "Prometheus text format"
// @Router       /metrics [get]
func (h *MetricsHandler) Prometheus(c *gin.Context) {
	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.Status(http.StatusOK)
	_ = h.reg.WritePrometheus(c.Writer)
}
