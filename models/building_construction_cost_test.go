package models

import (
	"testing"
	"time"
)

func TestBuildingConstructionCostModel(t *testing.T) {
	cost := BuildingConstructionCost{
		BuildingID: 1,
		ItemID:     2,
		Quantity:   10,
		CreatedAt:  time.Now(),
	}

	if cost.BuildingID != 1 {
		t.Errorf("Expected BuildingID to be 1, got %d", cost.BuildingID)
	}

	if cost.ItemID != 2 {
		t.Errorf("Expected ItemID to be 2, got %d", cost.ItemID)
	}

	if cost.Quantity != 10 {
		t.Errorf("Expected Quantity to be 10, got %d", cost.Quantity)
	}

	if cost.TableName() != "building_construction_costs" {
		t.Errorf("Expected TableName to be 'building_construction_costs', got '%s'", cost.TableName())
	}
}
