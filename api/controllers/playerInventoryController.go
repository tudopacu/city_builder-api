package controllers

import (
	"API/api/dto/requests"
	"API/api/services"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func GetPlayerInventory(c *gin.Context) {
	playerID, err := strconv.ParseUint(c.Param("player_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player_id"})
		return
	}

	inventoryDTOs, err := services.GetPlayerInventories(uint(playerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"player_inventories": inventoryDTOs})
}

func AddInventoryItem(c *gin.Context) {
	var request requests.AddInventoryItemRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	statusCode, response := services.AddInventoryItem(request)
	c.JSON(statusCode, response)
}
