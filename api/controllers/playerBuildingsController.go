package controllers

import (
	"API/api/dto/requests"
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func GetPlayerBuildings(c *gin.Context) {
	playerId, ok := requirePlayerID(c)
	if !ok {
		return
	}

	mapId, err := strconv.ParseUint(c.Param("map_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid map_id"})
		return
	}

	playerBuildingDTOs, err := services.GetPlayerBuildings(playerId, uint(mapId))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player_buildings": playerBuildingDTOs})
}

func GetPlayerPopulation(c *gin.Context) {
	playerId, ok := requirePlayerID(c)
	if !ok {
		return
	}

	mapId, err := strconv.ParseUint(c.Param("map_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid map_id"})
		return
	}

	population, err := services.GetPlayerPopulation(playerId, uint(mapId))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"population": population})
}

func DeletePlayerBuilding(c *gin.Context) {
	playerID, ok := requirePlayerID(c)
	if !ok {
		return
	}

	playerBuildingID, err := strconv.ParseUint(c.Param("player_building_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_building_id"})
		return
	}

	statusCode, serviceErr := services.DeletePlayerBuilding(playerID, uint(playerBuildingID))
	if serviceErr != nil {
		c.JSON(statusCode, gin.H{"error": serviceErr.Error()})
		return
	}

	c.JSON(statusCode, gin.H{"message": "player building deleted successfully"})
}

func AddPlayerBuilding(c *gin.Context) {
	playerID, ok := requirePlayerID(c)
	if !ok {
		return
	}

	var request requests.AddBuildingRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	statusCode, response := services.AddPlayerBuilding(playerID, request)
	c.JSON(statusCode, response)
}
