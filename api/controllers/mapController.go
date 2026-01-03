package controllers

import (
	"API/api/services"
	"API/database"
	"API/models"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

//TODO: move queries to service layer

func GetMaps(c *gin.Context) {
	var maps []models.Map

	if err := database.DB.Find(&maps).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no maps found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"maps": maps})
}

func GetMap(c *gin.Context) {
	mapId, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid map_id"})
		return
	}

	mapDto, err := services.GetMapByID(uint(mapId))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{"map": mapDto})
}

func GetTiles(c *gin.Context) {
	tiles, err := services.GetTiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, tiles)
}
