package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ApiResponse is the single response envelope every JSON endpoint uses.
// Code carries a stable, machine readable reason (for example "skip_locked")
// so the frontend never has to match on human readable copy.
type ApiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func SuccessResponse(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, ApiResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ApiResponse{
		Success: false,
		Message: message,
	})
}

// ErrorResponseCode is ErrorResponse with a machine readable code and an
// optional data payload. The five second Skip lock uses it to hand back the
// remaining milliseconds instead of a bare conflict.
func ErrorResponseCode(c *gin.Context, statusCode int, code, message string, data any) {
	c.JSON(statusCode, ApiResponse{
		Success: false,
		Code:    code,
		Message: message,
		Data:    data,
	})
}
