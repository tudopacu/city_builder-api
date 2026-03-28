package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

const DefaultInventoryCapacity = 100

func GetPlayerInventory(playerId uint) (dto.PlayerInventory, error) {
	var playerItems []models.PlayerItem

	if err := database.DB.
		Preload("Item").
		Find(&playerItems, "player_id = ?", playerId).
		Error; err != nil {

		log.Default().Printf("failed to fetch player items for player_id %d: %s", playerId, err)
		return dto.PlayerInventory{}, fmt.Errorf("failed to fetch player inventory for player_id %d", playerId)
	}

	// Convert models to DTOs
	items := make([]dto.PlayerInventoryItem, 0, len(playerItems))
	totalItems := 0
	for _, playerItem := range playerItems {
		items = append(items, playerItem.ToDTO())
		totalItems += playerItem.Quantity
	}

	return dto.PlayerInventory{
		PlayerID:      playerId,
		Items:         items,
		TotalCapacity: DefaultInventoryCapacity,
		TotalItems:    totalItems,
	}, nil
}
