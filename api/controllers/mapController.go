package controllers

import (
	"API/database"
	"API/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func GetMaps(c *gin.Context) {
	var maps []models.Map

	if err := database.DB.Find(&maps).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no maps found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"maps": maps})
}

func GetMap(c *gin.Context) {
	id := c.Param("id")
	var mapModel models.Map

	if err := database.DB.Preload("Terrains.Tile").First(&mapModel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "map not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"map": mapModel})
}
