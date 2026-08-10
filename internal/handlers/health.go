package handlers

import (
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

// HealthCheck godoc
// @Summary Show the status of the API
// @Description get the status of the API
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
