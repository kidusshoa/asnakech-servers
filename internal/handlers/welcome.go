package handlers

import (
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/gin-gonic/gin"
)

type WelcomeHandler struct {
	version string
}

func NewWelcomeHandler(version string) *WelcomeHandler {
	return &WelcomeHandler{version: version}
}

// Welcome godoc
// @Summary API welcome
// @Description Returns a short welcome payload for the v1 API root
// @Tags system
// @Produce json
// @Success 200 {object} response.Envelope
// @Router / [get]
func (h *WelcomeHandler) Welcome(c *gin.Context) {
	response.OK(c, gin.H{
		"message": "Welcome to Asnakech School API",
		"version": h.version,
	})
}
