package controllers

import (
	"API/api/dto/requests"
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func GetPlayerBuildings(c *gin.Context) {
	playerId, err := strconv.ParseUint(c.Param("player_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_id"})
		return
	}

	mapId, err := strconv.ParseUint(c.Param("map_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid map_id"})
		return
	}

	playerBuildingDTOs, err := services.GetPlayerBuildings(uint(playerId), uint(mapId))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player_buildings": playerBuildingDTOs})
}

func DeletePlayerBuilding(c *gin.Context) {
	playerBuildingID, err := strconv.ParseUint(c.Param("player_building_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_building_id"})
		return
	}

	statusCode, serviceErr := services.DeletePlayerBuilding(uint(playerBuildingID))
	if serviceErr != nil {
		c.JSON(statusCode, gin.H{"error": serviceErr.Error()})
		return
	}

	c.JSON(statusCode, gin.H{"message": "player building deleted successfully"})
}

func AddPlayerBuilding(c *gin.Context) {
	var request requests.AddBuildingRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	statusCode, response := services.AddPlayerBuilding(request)
	c.JSON(statusCode, response)
}
