package services

import (
	"API/models"
	"testing"
)

func TestTotalInventoryQuantity(t *testing.T) {
	items := []models.PlayerInventoryItem{
		{Quantity: 10},
		{Quantity: 25},
		{Quantity: 5},
	}

	total := totalInventoryQuantity(items)
	if total != 40 {
		t.Errorf("Expected total quantity to be 40, got %d", total)
	}
}

func TestTotalInventoryQuantityEmpty(t *testing.T) {
	items := []models.PlayerInventoryItem{}

	total := totalInventoryQuantity(items)
	if total != 0 {
		t.Errorf("Expected total quantity to be 0, got %d", total)
	}
}
