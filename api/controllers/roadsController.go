package controllers

import (
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func GetRoads(c *gin.Context) {
	playerID, ok := requirePlayerID(c)
	if !ok {
		return
	}

	mapID, err := strconv.ParseUint(c.Param("map_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid map_id"})
		return
	}

	roads, err := services.GetRoadsByPlayerAndMap(playerID, uint(mapID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roads": roads})
}
