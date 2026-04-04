package models

import (
	"testing"
	"time"
)

func TestPlayerInventoryModel(t *testing.T) {
	inventory := PlayerInventory{
		PlayerID:         1,
		PlayerBuildingID: 2,
		Capacity:         100,
		CreatedAt:        time.Now(),
	}

	if inventory.PlayerID != 1 {
		t.Errorf("Expected PlayerID to be 1, got %d", inventory.PlayerID)
	}

	if inventory.PlayerBuildingID != 2 {
		t.Errorf("Expected PlayerBuildingID to be 2, got %d", inventory.PlayerBuildingID)
	}

	if inventory.Capacity != 100 {
		t.Errorf("Expected Capacity to be 100, got %d", inventory.Capacity)
	}

	if inventory.TableName() != "player_inventories" {
		t.Errorf("Expected TableName to be 'player_inventories', got '%s'", inventory.TableName())
	}
}

func TestPlayerInventoryItemToDTO(t *testing.T) {
	item := PlayerInventoryItem{
		ID:       5,
		Quantity: 10,
		Item: Item{
			ID:   3,
			Name: "Wood",
			Type: "resource",
		},
	}

	itemDTO := item.ToDTO()

	if itemDTO.ID != 5 {
		t.Errorf("Expected ID to be 5, got %d", itemDTO.ID)
	}

	if itemDTO.Quantity != 10 {
		t.Errorf("Expected Quantity to be 10, got %d", itemDTO.Quantity)
	}

	if itemDTO.ItemID != 3 {
		t.Errorf("Expected ItemID to be 3, got %d", itemDTO.ItemID)
	}
}

func TestPlayerInventoryItemModel(t *testing.T) {
	item := PlayerInventoryItem{
		PlayerInventoryID: 1,
		ItemID:            3,
		Quantity:          10,
		CreatedAt:         time.Now(),
	}

	if item.PlayerInventoryID != 1 {
		t.Errorf("Expected PlayerInventoryID to be 1, got %d", item.PlayerInventoryID)
	}

	if item.ItemID != 3 {
		t.Errorf("Expected ItemID to be 3, got %d", item.ItemID)
	}

	if item.Quantity != 10 {
		t.Errorf("Expected Quantity to be 10, got %d", item.Quantity)
	}

	if item.TableName() != "player_inventory_items" {
		t.Errorf("Expected TableName to be 'player_inventory_items', got '%s'", item.TableName())
	}
}

func TestPlayerInventoryToDTO(t *testing.T) {
	inventory := PlayerInventory{
		ID:       1,
		Capacity: 50,
		PlayerBuilding: PlayerBuilding{
			ID: 2,
		},
		InventoryItems: []PlayerInventoryItem{
			{
				ID:       10,
				Quantity: 5,
				Item: Item{
					ID:   3,
					Name: "Stone",
					Type: "resource",
				},
			},
		},
	}

	inventoryDTO := inventory.ToDTO()

	if inventoryDTO.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", inventoryDTO.ID)
	}

	if inventoryDTO.Capacity != 50 {
		t.Errorf("Expected Capacity to be 50, got %d", inventoryDTO.Capacity)
	}

	if inventoryDTO.PlayerBuildingID != 2 {
		t.Errorf("Expected PlayerBuilding.ID to be 2, got %d", inventoryDTO.PlayerBuildingID)
	}

	if len(inventoryDTO.Items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(inventoryDTO.Items))
	}

	if inventoryDTO.Items[0].ID != 10 {
		t.Errorf("Expected Items[0].ID to be 10, got %d", inventoryDTO.Items[0].ID)
	}

	if inventoryDTO.Items[0].ItemID != 3 {
		t.Errorf("Expected Items[0].ItemID to be 3, got '%d'", inventoryDTO.Items[0].ItemID)
	}
}
