package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

func GetPlayerInventories(playerID uint) ([]dto.PlayerInventory, error) {
	var inventories []models.PlayerInventory

	if err := database.DB.
		Preload("PlayerBuilding").
		Preload("PlayerBuilding.Building").
		Preload("PlayerBuilding.Building.Category").
		Preload("PlayerBuilding.BuildingLevel").
		Preload("InventoryItems").
		Preload("InventoryItems.Item").
		Find(&inventories, "player_id = ?", playerID).
		Error; err != nil {

		log.Default().Printf("failed to fetch player inventories for player_id %d: %s", playerID, err)
		return []dto.PlayerInventory{}, fmt.Errorf("failed to fetch player inventories for player_id %d", playerID)
	}

	inventoryDTOs := make([]dto.PlayerInventory, 0, len(inventories))
	for _, inventory := range inventories {
		inventoryDTOs = append(inventoryDTOs, inventory.ToDTO())
	}

	return inventoryDTOs, nil
}
