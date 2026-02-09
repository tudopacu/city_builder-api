package models

import (
	"testing"
	"time"
)

func TestPlayerItemModel(t *testing.T) {
	playerItem := PlayerItem{
		ID:        1,
		PlayerID:  100,
		ItemID:    200,
		Quantity:  50,
		CreatedAt: time.Now(),
	}

	if playerItem.ID != 1 {
		t.Errorf("Expected ID to be 1, got %d", playerItem.ID)
	}

	if playerItem.PlayerID != 100 {
		t.Errorf("Expected PlayerID to be 100, got %d", playerItem.PlayerID)
	}

	if playerItem.ItemID != 200 {
		t.Errorf("Expected ItemID to be 200, got %d", playerItem.ItemID)
	}

	if playerItem.Quantity != 50 {
		t.Errorf("Expected Quantity to be 50, got %d", playerItem.Quantity)
	}

	if playerItem.TableName() != "player_items" {
		t.Errorf("Expected TableName to be 'player_items', got '%s'", playerItem.TableName())
	}
}

func TestPlayerItemToDTO(t *testing.T) {
	item := Item{
		ID:   200,
		Name: "Wood",
		Type: "resource",
	}

	playerItem := PlayerItem{
		ID:       1,
		PlayerID: 100,
		ItemID:   200,
		Quantity: 50,
		Item:     item,
	}

	dto := playerItem.ToDTO()

	if dto.ItemID != 200 {
		t.Errorf("Expected DTO ItemID to be 200, got %d", dto.ItemID)
	}

	if dto.Name != "Wood" {
		t.Errorf("Expected DTO Name to be 'Wood', got '%s'", dto.Name)
	}

	if dto.Type != "resource" {
		t.Errorf("Expected DTO Type to be 'resource', got '%s'", dto.Type)
	}

	if dto.Quantity != 50 {
		t.Errorf("Expected DTO Quantity to be 50, got %d", dto.Quantity)
	}
}
