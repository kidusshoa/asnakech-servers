package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/asnakech/asnakech-servers/internal/platform/ready"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	version string
	checker *ready.Checker
}

func NewHealthHandler(version string, checker *ready.Checker) *HealthHandler {
	return &HealthHandler{
		version: version,
		checker: checker,
	}
}

// HealthCheck godoc
// @Summary Liveness probe
// @Description Returns OK when the process is running
// @Tags health
// @Produce json
// @Success 200 {object} response.Envelope
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "ok",
		"version": h.version,
	})
}

// ReadyCheck godoc
// @Summary Readiness probe
// @Description Returns OK when configured dependencies are reachable
// @Tags health
// @Produce json
// @Success 200 {object} response.Envelope
// @Failure 503 {object} response.Envelope
// @Router /ready [get]
func (h *HealthHandler) ReadyCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	statuses, ok := h.checker.Check(ctx)
	payload := gin.H{
		"status":  statusLabel(ok),
		"version": h.version,
		"checks":  statuses,
	}

	if !ok {
		c.JSON(http.StatusServiceUnavailable, response.Envelope{
			Success: false,
			Data:    payload,
			Error: &response.ErrorBody{
				Code:    "not_ready",
				Message: "one or more dependencies are unavailable",
			},
			Meta: response.Meta{RequestID: c.GetString("request_id")},
		})
		return
	}

	response.OK(c, payload)
}

func statusLabel(ok bool) string {
	if ok {
		return "ready"
	}
	return "not_ready"
}
