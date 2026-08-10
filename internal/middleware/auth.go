package middleware

import (
	"strings"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/asnakech/asnakech-servers/internal/auth"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/gin-gonic/gin"
)

const (
	ContextUserID   = "auth_user_id"
	ContextEmail    = "auth_email"
	ContextRoleCode = "auth_role_code"
)

// BearerAuth validates Authorization: Bearer <access_token>.
func BearerAuth(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Fail(c, apperr.Unauthorized("missing authorization header"))
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			response.Fail(c, apperr.Unauthorized("invalid authorization header"))
			c.Abort()
			return
		}

		claims, err := tokens.ParseAccessToken(parts[1])
		if err != nil {
			response.Fail(c, apperr.Unauthorized("invalid or expired access token"))
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextEmail, claims.Email)
		c.Set(ContextRoleCode, claims.RoleCode)
		c.Next()
	}
}
