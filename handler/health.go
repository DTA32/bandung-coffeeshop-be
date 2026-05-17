package handler

import (
	"github.com/dta32/bandung-coffeeshop-be/helper"
	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	helper.Success(c, gin.H{"status": "ok"})
}
