package controllers

import (
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
