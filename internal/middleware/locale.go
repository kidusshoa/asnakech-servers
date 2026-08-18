package middleware

import (
	"github.com/asnakech/asnakech-servers/internal/i18n"
	"github.com/gin-gonic/gin"
)

// Locale resolves locale from ?lang= or Accept-Language and stores it on the context.
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := i18n.Resolve(c.Query("lang"), c.GetHeader("Accept-Language"))
		c.Set(i18n.ContextLocale, locale)
		c.Header("Content-Language", locale)
		c.Next()
	}
}
