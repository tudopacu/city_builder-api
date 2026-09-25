package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetCurrentTimestamp(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now(),
	})
}
