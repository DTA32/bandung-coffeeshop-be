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
	c.JSON(200, successResponse{Success: true, Data: data})
}

func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, errorResponse{Success: false, Error: msg})
}
