package middleware

import (
	"strings"

	"github.com/asnakech/asnakech-servers/internal/auth"
	"github.com/gin-gonic/gin"
)

// OptionalBearerAuth parses a Bearer token when present; otherwise continues anonymously.
func OptionalBearerAuth(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.Next()
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			c.Next()
			return
		}
		claims, err := tokens.ParseAccessToken(parts[1])
		if err != nil {
			c.Next()
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRoleCode, claims.RoleCode)
		c.Next()
	}
}
