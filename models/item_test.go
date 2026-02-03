package models

import (
	"testing"
	"time"
)

func TestItemModel(t *testing.T) {
	item := Item{
		Name:        "Test Item",
		Type:        "resource",
		IsTradeable: true,
		CreatedAt:   time.Now(),
	}

	if item.Name != "Test Item" {
		t.Errorf("Expected Name to be 'Test Item', got '%s'", item.Name)
	}

	if item.Type != "resource" {
		t.Errorf("Expected Type to be 'resource', got '%s'", item.Type)
	}

	if item.IsTradeable != true {
		t.Errorf("Expected IsTradeable to be true, got %v", item.IsTradeable)
	}

	if item.TableName() != "items" {
		t.Errorf("Expected TableName to be 'items', got '%s'", item.TableName())
	}
}

func TestItemRecipeModel(t *testing.T) {
	recipe := ItemRecipe{
		ItemID:                1,
		ProductionTimeSeconds: 60,
		CreatedAt:             time.Now(),
	}

	if recipe.ItemID != 1 {
		t.Errorf("Expected ItemID to be 1, got %d", recipe.ItemID)
	}

	if recipe.ProductionTimeSeconds != 60 {
		t.Errorf("Expected ProductionTimeSeconds to be 60, got %d", recipe.ProductionTimeSeconds)
	}

	if recipe.TableName() != "item_recipes" {
		t.Errorf("Expected TableName to be 'item_recipes', got '%s'", recipe.TableName())
	}
}

func TestItemRecipeInputModel(t *testing.T) {
	input := ItemRecipeInput{
		RecipeID:    1,
		InputItemID: 2,
		Quantity:    5,
		CreatedAt:   time.Now(),
	}

	if input.RecipeID != 1 {
		t.Errorf("Expected RecipeID to be 1, got %d", input.RecipeID)
	}

	if input.InputItemID != 2 {
		t.Errorf("Expected InputItemID to be 2, got %d", input.InputItemID)
	}

	if input.Quantity != 5 {
		t.Errorf("Expected Quantity to be 5, got %d", input.Quantity)
	}

	if input.TableName() != "item_recipe_inputs" {
		t.Errorf("Expected TableName to be 'item_recipe_inputs', got '%s'", input.TableName())
	}
}
