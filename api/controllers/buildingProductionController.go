package controllers

import (
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func StartBuildingProduction(c *gin.Context) {
	playerID, err := strconv.ParseUint(c.Param("player_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_id"})
		return
	}

	playerBuildingID, err := strconv.ParseUint(c.Param("player_building_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_building_id"})
		return
	}

	buildingProductionID, err := strconv.ParseUint(c.Param("building_production_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building_production_id"})
		return
	}

	statusCode, response, serviceErr := services.StartProduction(uint(playerID), uint(playerBuildingID), uint(buildingProductionID))
	if serviceErr != nil {
		c.JSON(statusCode, gin.H{"error": serviceErr.Error()})
		return
	}

	c.JSON(statusCode, response)
}

func CollectBuildingProduction(c *gin.Context) {
	playerID, err := strconv.ParseUint(c.Param("player_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_id"})
		return
	}

	playerBuildingID, err := strconv.ParseUint(c.Param("player_building_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_building_id"})
		return
	}

	buildingProductionID, err := strconv.ParseUint(c.Param("building_production_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building_production_id"})
		return
	}

	statusCode, serviceErr := services.CollectProduction(uint(playerID), uint(playerBuildingID), uint(buildingProductionID))
	if serviceErr != nil {
		c.JSON(statusCode, gin.H{"error": serviceErr.Error()})
		return
	}

	c.JSON(statusCode, gin.H{"message": "production collected successfully"})
}
