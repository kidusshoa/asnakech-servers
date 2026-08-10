package response

import (
	"net/http"

	"github.com/asnakech/asnakech-servers/internal/apperr"
	"github.com/gin-gonic/gin"
)

// Envelope is the standard JSON shape returned by the API.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorBody  `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// ErrorBody is the client-facing error payload.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta holds optional response metadata (pagination, request info, etc.).
type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	Page      int    `json:"page,omitempty"`
	PerPage   int    `json:"per_page,omitempty"`
	Total     int64  `json:"total,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, data, nil)
}

func Created(c *gin.Context, data interface{}) {
	JSON(c, http.StatusCreated, data, nil)
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func JSON(c *gin.Context, status int, data interface{}, meta interface{}) {
	c.JSON(status, Envelope{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func Fail(c *gin.Context, err error) {
	appErr, ok := apperr.As(err)
	if !ok {
		appErr = apperr.Internal("an unexpected error occurred")
	}

	c.JSON(appErr.HTTPStatus, Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    string(appErr.Code),
			Message: appErr.Message,
		},
		Meta: Meta{RequestID: c.GetString("request_id")},
	})
}

func FailCode(c *gin.Context, status int, code apperr.Code, message string) {
	c.JSON(status, Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:    string(code),
			Message: message,
		},
		Meta: Meta{RequestID: c.GetString("request_id")},
	})
}
