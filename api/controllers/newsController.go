package controllers

import (
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetNews(c *gin.Context) {
	news, err := services.GetNews()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"news": news})
}
