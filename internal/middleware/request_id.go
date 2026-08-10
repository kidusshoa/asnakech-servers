package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	headerRequestID = "X-Request-ID"
	contextKeyRID   = "request_id"
)

// RequestID ensures every request has an ID: reuse the client header when
// present, otherwise generate one. The value is stored on the Gin context
// and echoed back on the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(headerRequestID)
		if rid == "" {
			rid = newRequestID()
		}

		c.Set(contextKeyRID, rid)
		c.Writer.Header().Set(headerRequestID, rid)
		c.Next()
	}
}

func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}
