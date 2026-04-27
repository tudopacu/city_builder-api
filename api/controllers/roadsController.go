package controllers

import (
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetRoads(c *gin.Context) {
	roads, err := services.GetAllRoads()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roads": roads})
}
