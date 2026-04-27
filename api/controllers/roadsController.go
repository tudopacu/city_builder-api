package controllers

import (
	"API/api/dto/requests"
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func GetRoads(c *gin.Context) {
	playerID, err := strconv.ParseUint(c.Param("player_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_id"})
		return
	}

	mapID, err := strconv.ParseUint(c.Param("map_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid map_id"})
		return
	}

	roads, err := services.GetRoadsByPlayerAndMap(uint(playerID), uint(mapID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roads": roads})
}

func AddRoads(c *gin.Context) {
	var request requests.AddRoadsRequest

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	statusCode, response := services.AddRoads(request)
	c.JSON(statusCode, response)
}
