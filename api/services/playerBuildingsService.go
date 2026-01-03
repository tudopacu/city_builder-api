package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

func GetPlayerBuildings(playerId uint, mapId uint) ([]dto.PlayerBuilding, error) {
	var playerBuildingModels []models.PlayerBuilding

	if err := database.DB.
		Preload("Building").
		Preload("Building.Category").
		Preload("Building.Levels").
		Preload("BuildingLevel").
		Find(&playerBuildingModels, "player_id = ? AND map_id = ?", playerId, mapId).
		Error; err != nil {

		log.Default().Println(fmt.Sprintf(
			"failed to fetch player buildings for player_id %d on map_id %d",
			playerId, mapId), err)

		return []dto.PlayerBuilding{}, fmt.Errorf(
			"failed to fetch player buildings for player_id %d on map_id %d",
			playerId, mapId,
		)
	}

	playerBuildingDTOs := make([]dto.PlayerBuilding, 0, len(playerBuildingModels))
	for _, playerBuilding := range playerBuildingModels {
		playerBuildingDTOs = append(playerBuildingDTOs, playerBuilding.ToDTO())
	}

	return playerBuildingDTOs, nil
}
