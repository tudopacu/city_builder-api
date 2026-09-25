package services

import (
	"API/api/dto"
	"API/api/dto/responses"
	"API/database"
	"API/models"
	"fmt"
	"log"
	"net/http"
	"time"
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

	var existing models.BuildingCurrentProduction
	err := database.DB.Where(
		"player_id = ? AND player_building_id = ? AND building_production_id = ? AND status IN ('PENDING','DONE')",
		playerID, playerBuildingID, buildingProductionID,
	).First(&existing).Error
	if err == nil {
		return http.StatusConflict, nil, fmt.Errorf("production already in progress")
	}

	currentTimestamp := time.Now()
	endTime := currentTimestamp.Add(time.Duration(buildingProduction.ProductionTimeSeconds) * time.Second)

	entry := models.BuildingCurrentProduction{
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
	var entry models.BuildingCurrentProduction
	err := database.DB.Where(
		"player_id = ? AND player_building_id = ? AND building_production_id = ? AND status IN ('PENDING','DONE')",
		playerID, playerBuildingID, buildingProductionID,
	).First(&entry).Error

	if err != nil {
		log.Default().Printf("building current production not found for player_id %d, player_building_id %d, building_production_id %d: %s",
			playerID, playerBuildingID, buildingProductionID, err)
		return http.StatusNotFound, fmt.Errorf("production not found")
	}

	if time.Now().Before(entry.EndTime) {
		return http.StatusBadRequest, fmt.Errorf("production is not finished yet")
	}

	if err := database.DB.Model(&entry).Update("status", "COLLECTED").Error; err != nil {
		log.Default().Printf("failed to update production status: %s", err)
		return http.StatusInternalServerError, fmt.Errorf("failed to collect production")
	}

	return http.StatusOK, nil
}

func DeleteCollectedProductions() {
	result := database.DB.Where("status = ?", "COLLECTED").Delete(&models.BuildingCurrentProduction{})
	if result.Error != nil {
		log.Default().Printf("failed to delete collected productions: %s", result.Error)
		return
	}
	log.Default().Printf("deleted %d collected productions", result.RowsAffected)
}
