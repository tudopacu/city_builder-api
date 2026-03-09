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
