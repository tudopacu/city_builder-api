package controllers

import (
	"API/api/dto/requests"
	"API/api/dto/responses"
	"API/api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetPlayerInventories(c *gin.Context) {
	playerID, ok := requirePlayerID(c)
	if !ok {
		return
	}

	mapID, err := strconv.ParseUint(c.Param("map_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid map_id"})
		return
	}

	inventoryDTOs, totalQuantity, totalCapacity, err := services.GetPlayerInventories(playerID, uint(mapID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player_inventory": responses.PlayerInventoryResponse{PlayerInventories: inventoryDTOs, TotalQuantity: totalQuantity, TotalCapacity: totalCapacity}})
}

func AddInventoryItem(c *gin.Context) {
	playerID, ok := requirePlayerID(c)
	if !ok {
		return
	}

	var request requests.AddInventoryItemRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	statusCode, response := services.AddInventoryItem(playerID, request)
	c.JSON(statusCode, response)
}
