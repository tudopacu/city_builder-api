package controllers

import (
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetItems(c *gin.Context) {
	items, err := services.GetAllItems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}
