package models

import (
	"testing"
	"time"
)

func TestBuildingProductionModel(t *testing.T) {
	production := BuildingProduction{
		BuildingID:            1,
		ItemID:                2,
		ProductionTimeSeconds: 120,
		Quantity:              5,
		CreatedAt:             time.Now(),
	}

	if production.BuildingID != 1 {
		t.Errorf("Expected BuildingID to be 1, got %d", production.BuildingID)
	}

	if production.ItemID != 2 {
		t.Errorf("Expected ItemID to be 2, got %d", production.ItemID)
	}

	if production.ProductionTimeSeconds != 120 {
		t.Errorf("Expected ProductionTimeSeconds to be 120, got %d", production.ProductionTimeSeconds)
	}

	if production.Quantity != 5 {
		t.Errorf("Expected Quantity to be 5, got %d", production.Quantity)
	}

	if production.TableName() != "building_productions" {
		t.Errorf("Expected TableName to be 'building_productions', got '%s'", production.TableName())
	}
}
