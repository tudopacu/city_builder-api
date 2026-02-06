package controllers

import (
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetBuildings(c *gin.Context) {
	buildings, err := services.GetAllBuildings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"buildings": buildings})
}
