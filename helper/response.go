package helper

import "github.com/gin-gonic/gin"

type successResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

type errorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func Success(c *gin.Context, data any) {
	SuccessStatus(c, 200, data)
}

// SuccessStatus is Success with a non-200 status, e.g. 201 Created.
func SuccessStatus(c *gin.Context, code int, data any) {
	c.JSON(code, successResponse{Success: true, Data: data})
}

func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, errorResponse{Success: false, Error: msg})
}
