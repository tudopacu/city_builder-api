package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"API/api/dto/responses"
	"API/database"
	"API/models"
	"fmt"
	"log"
	"net/http"
)

func GetPlayerInventories(playerID uint) ([]dto.PlayerInventory, int, int, error) {
	var inventories []models.PlayerInventory

	if err := database.DB.
		Preload("InventoryItems").
		Preload("InventoryItems.Item").
		Find(&inventories, "player_id = ?", playerID).
		Error; err != nil {

		log.Default().Printf("failed to fetch player inventories for player_id %d: %s", playerID, err)
		return []dto.PlayerInventory{}, 0, 0, fmt.Errorf("failed to fetch player inventories for player_id %d", playerID)
	}

	inventoryDTOs := make([]dto.PlayerInventory, 0, len(inventories))
	for _, inventory := range inventories {
		inventoryDTOs = append(inventoryDTOs, inventory.ToDTO())
	}

	totalQuantity, totalCapacity := CalculateTotalInventoryStats(inventoryDTOs)

	return inventoryDTOs, totalQuantity, totalCapacity, nil
}

func loadInventoryWithAssociations(inventoryID uint) (*models.PlayerInventory, error) {
	var inventory models.PlayerInventory
	if err := database.DB.
		Preload("PlayerBuilding").
		Preload("PlayerBuilding.Building").
		Preload("PlayerBuilding.Building.Category").
		Preload("PlayerBuilding.BuildingLevel").
		Preload("InventoryItems").
		Preload("InventoryItems.Item").
		First(&inventory, inventoryID).Error; err != nil {
		log.Default().Printf("failed to load inventory with ID %d: %s", inventoryID, err)
		return nil, fmt.Errorf("inventory not found")
	}
	return &inventory, nil
}

func totalInventoryQuantity(items []models.PlayerInventoryItem) int {
	total := 0
	for _, item := range items {
		total += item.Quantity
	}
	return total
}

func AddInventoryItem(request requests.AddInventoryItemRequest) (int, responses.AddInventoryItemResponse) {
	// Load inventory and verify it belongs to the player
	inventory, err := loadInventoryWithAssociations(request.InventoryID)
	if err != nil {
		return http.StatusNotFound, responses.AddInventoryItemResponse{Error: "inventory not found"}
	}

	if inventory.PlayerID != request.PlayerID {
		return http.StatusForbidden, responses.AddInventoryItemResponse{Error: "inventory does not belong to player"}
	}

	// Validate the item exists
	var item models.Item
	if err := database.DB.First(&item, request.ItemID).Error; err != nil {
		log.Default().Printf("item not found, item_id %d: %s", request.ItemID, err)
		return http.StatusNotFound, responses.AddInventoryItemResponse{Error: "item not found"}
	}

	// Check capacity: sum existing quantities and compare against inventory capacity
	currentTotal := totalInventoryQuantity(inventory.InventoryItems)

	// Find existing inventory item for this item (if any).
	var existingEntry *models.PlayerInventoryItem
	for i := range inventory.InventoryItems {
		if inventory.InventoryItems[i].ItemID == request.ItemID {
			existingEntry = &inventory.InventoryItems[i]
			break
		}
	}

	if currentTotal+request.Quantity > inventory.Capacity {
		return http.StatusBadRequest, responses.AddInventoryItemResponse{
			Error: fmt.Sprintf("not enough capacity: available %d, requested %d", inventory.Capacity-currentTotal, request.Quantity),
		}
	}

	// Save: update existing entry or create a new one
	if existingEntry != nil {
		existingEntry.Quantity += request.Quantity
		if err := database.DB.Save(existingEntry).Error; err != nil {
			log.Default().Printf("failed to update inventory item for inventory_id %d, item_id %d: %s", request.InventoryID, request.ItemID, err)
			return http.StatusInternalServerError, responses.AddInventoryItemResponse{Error: "failed to update inventory item"}
		}
	} else {
		newEntry := models.PlayerInventoryItem{
			PlayerInventoryID: request.InventoryID,
			ItemID:            request.ItemID,
			Quantity:          request.Quantity,
		}
		if err := database.DB.Create(&newEntry).Error; err != nil {
			log.Default().Printf("failed to create inventory item for inventory_id %d, item_id %d: %s", request.InventoryID, request.ItemID, err)
			return http.StatusInternalServerError, responses.AddInventoryItemResponse{Error: "failed to add inventory item"}
		}
	}

	// Return the updated inventory
	updatedInventory, err := loadInventoryWithAssociations(request.InventoryID)
	if err != nil {
		return http.StatusInternalServerError, responses.AddInventoryItemResponse{Error: "failed to reload inventory"}
	}

	inventoryDTO := updatedInventory.ToDTO()
	return http.StatusOK, responses.AddInventoryItemResponse{PlayerInventory: &inventoryDTO}
}

func CalculateTotalInventoryStats(inventories []dto.PlayerInventory) (totalQuantity int, totalCapacity int) {
	for _, inventory := range inventories {
		totalCapacity += inventory.Capacity
		for _, item := range inventory.Items {
			totalQuantity += item.Quantity
		}
	}
	return totalQuantity, totalCapacity
}
