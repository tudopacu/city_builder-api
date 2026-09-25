package services

import (
	"API/api/dto"
	"API/api/dto/responses"
	"API/database"
	"API/models"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func StartProduction(playerID, playerBuildingID, buildingProductionID uint) (int, *responses.StartProductionResponse, error) {
	var playerBuilding models.PlayerBuilding
	if err := database.DB.First(&playerBuilding, playerBuildingID).Error; err != nil {
		log.Default().Printf("player building not found, id %d: %s", playerBuildingID, err)
		return http.StatusNotFound, nil, fmt.Errorf("player building not found")
	}

	var buildingProduction models.BuildingProduction
	if err := database.DB.Preload("Item").First(&buildingProduction, buildingProductionID).Error; err != nil {
		log.Default().Printf("building production not found, id %d: %s", buildingProductionID, err)
		return http.StatusNotFound, nil, fmt.Errorf("building production not found")
	}

	if buildingProduction.BuildingID != playerBuilding.BuildingID {
		return http.StatusBadRequest, nil, fmt.Errorf("building production does not belong to this building")
	}

	var existing models.PlayerBuildingProduction
	err := database.DB.Where(
		"player_id = ? AND player_building_id = ? AND building_production_id = ? AND status IN ('PENDING','DONE')",
		playerID, playerBuildingID, buildingProductionID,
	).First(&existing).Error
	if err == nil {
		return http.StatusConflict, nil, fmt.Errorf("production already in progress")
	}

	currentTimestamp := time.Now()
	endTime := currentTimestamp.Add(time.Duration(buildingProduction.ProductionTimeSeconds) * time.Second)

	entry := models.PlayerBuildingProduction{
		PlayerID:             playerID,
		PlayerBuildingID:     playerBuildingID,
		BuildingProductionID: buildingProductionID,
		EndTime:              endTime,
		Status:               "PENDING",
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		log.Default().Printf("failed to create building current production: %s", err)
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to start production")
	}

	return http.StatusCreated, &responses.StartProductionResponse{
		BuildingCurrentProduction: &dto.BuildingCurrentProduction{
			ID:                   entry.ID,
			EndTime:              entry.EndTime,
			Status:               entry.Status,
			BuildingProductionID: entry.BuildingProductionID,
			ItemName:             buildingProduction.Item.Name,
			Quantity:             buildingProduction.Quantity,
		},
		CurrentTimestamp: currentTimestamp,
	}, nil
}

func CollectProduction(playerID, playerBuildingID, buildingProductionID uint) (int, error) {
	statusCode := http.StatusInternalServerError
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the production so concurrent collection requests cannot award it twice.
		var entry models.PlayerBuildingProduction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("BuildingProduction").
			Where(
				"player_id = ? AND player_building_id = ? AND building_production_id = ? AND status IN ('PENDING','DONE')",
				playerID, playerBuildingID, buildingProductionID,
			).First(&entry).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				statusCode = http.StatusNotFound
				return fmt.Errorf("production not found")
			}
			return fmt.Errorf("failed to load production: %w", err)
		}

		if time.Now().Before(entry.EndTime) {
			statusCode = http.StatusBadRequest
			return fmt.Errorf("production is not finished yet")
		}

		production := entry.BuildingProduction
		if production.Quantity <= 0 {
			return fmt.Errorf("invalid production quantity: %d", production.Quantity)
		}

		// Lock inventories in a consistent order, and use locking reads for their
		// contents so capacity reflects any collection that committed while waiting.
		var inventories []models.PlayerInventory
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("InventoryItems", func(db *gorm.DB) *gorm.DB {
				return db.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id ASC")
			}).
			Where("player_id = ?", playerID).
			Order("id ASC").
			Find(&inventories).Error; err != nil {
			return fmt.Errorf("failed to load player inventories: %w", err)
		}

		availableCapacity := make([]int, len(inventories))
		totalAvailable := 0
		for i, inventory := range inventories {
			availableCapacity[i] = max(0, inventory.Capacity-totalInventoryQuantity(inventory.InventoryItems))
			totalAvailable += availableCapacity[i]
		}

		// Check the entire batch before writing any inventory items.
		if totalAvailable < production.Quantity {
			statusCode = http.StatusBadRequest
			return fmt.Errorf("not enough capacity: available %d, requested %d", totalAvailable, production.Quantity)
		}

		remaining := production.Quantity
		for i, inventory := range inventories {
			if remaining == 0 {
				break
			}
			quantity := min(remaining, availableCapacity[i])
			if quantity == 0 {
				continue
			}

			var existingItem *models.PlayerInventoryItem
			for j := range inventory.InventoryItems {
				if inventory.InventoryItems[j].ItemID == production.ItemID {
					existingItem = &inventory.InventoryItems[j]
					break
				}
			}

			if existingItem != nil {
				if err := tx.Model(existingItem).
					Update("quantity", gorm.Expr("quantity + ?", quantity)).Error; err != nil {
					return fmt.Errorf("failed to update inventory item: %w", err)
				}
			} else {
				item := models.PlayerInventoryItem{
					PlayerInventoryID: inventory.ID,
					ItemID:            production.ItemID,
					Quantity:          quantity,
				}
				if err := tx.Create(&item).Error; err != nil {
					return fmt.Errorf("failed to create inventory item: %w", err)
				}
			}
			remaining -= quantity
		}

		if err := tx.Model(&models.PlayerBuildingProduction{}).
			Where("id = ?", entry.ID).
			Update("status", "COLLECTED").Error; err != nil {
			return fmt.Errorf("failed to update production status: %w", err)
		}
		return nil
	})

	if err != nil {
		if statusCode == http.StatusInternalServerError {
			log.Default().Printf("failed to collect production for player_id %d, player_building_id %d, building_production_id %d: %s",
				playerID, playerBuildingID, buildingProductionID, err)
			return statusCode, fmt.Errorf("failed to collect production")
		}
		return statusCode, err
	}

	return http.StatusOK, nil
}

func DeleteCollectedProductions() {
	result := database.DB.Where("status = ?", "COLLECTED").Delete(&models.PlayerBuildingProduction{})
	if result.Error != nil {
		log.Default().Printf("failed to delete collected productions: %s", result.Error)
		return
	}
	log.Default().Printf("deleted %d collected productions", result.RowsAffected)
}
