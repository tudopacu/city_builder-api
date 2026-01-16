package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"API/database"
	"API/models"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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

func AddPlayerBuilding(request requests.AddBuildingRequest) (int, gin.H) {
	// Validate coordinates are non-negative
	if request.X < 0 || request.Y < 0 {
		log.Default().Println(fmt.Sprintf("invalid coordinates: x=%d, y=%d", request.X, request.Y))
		return http.StatusBadRequest, gin.H{"error": "coordinates must be non-negative"}
	}

	// 1. Validate building exists and get its dimensions
	var building models.Building
	if err := database.DB.First(&building, request.BuildingID).Error; err != nil {
		log.Default().Println(fmt.Sprintf("building not found, building_id %d", request.BuildingID), err)
		return http.StatusNotFound, gin.H{"error": "building not found"}
	}

	// 2. Get the first building level (level 1) for the building
	var buildingLevel models.BuildingLevel
	if err := database.DB.Where("building_id = ? AND level = ?", request.BuildingID, 1).First(&buildingLevel).Error; err != nil {
		log.Default().Println(fmt.Sprintf("building level 1 not found for building_id %d", request.BuildingID), err)
		return http.StatusNotFound, gin.H{"error": "building level not found"}
	}

	// 3. Check if tiles at the building position are grass or dirt
	// Build list of coordinates to check
	var coordinates []struct {
		X int
		Y int
	}
	for x := request.X; x < request.X+building.Width; x++ {
		for y := request.Y; y < request.Y+building.Length; y++ {
			coordinates = append(coordinates, struct {
				X int
				Y int
			}{X: x, Y: y})
		}
	}

	// Fetch all terrains in one query
	var terrains []models.Terrain
	query := database.DB.Preload("Tile").Where("map_id = ?", request.MapID)
	
	// Build OR conditions for all coordinates
	orConditions := database.DB.Where("1 = 0") // Start with false condition
	for _, coord := range coordinates {
		orConditions = orConditions.Or("(x = ? AND y = ?)", coord.X, coord.Y)
	}
	
	if err := query.Where(orConditions).Find(&terrains).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to fetch terrains for building placement on map_id %d", request.MapID), err)
		return http.StatusInternalServerError, gin.H{"error": "failed to validate terrain"}
	}

	// Verify we found all expected tiles
	if len(terrains) != len(coordinates) {
		log.Default().Println(fmt.Sprintf("not all terrain tiles found for building placement, expected %d, found %d", len(coordinates), len(terrains)))
		return http.StatusBadRequest, gin.H{"error": "some terrain tiles are missing"}
	}

	// Validate all tiles are grass or dirt
	for _, terrain := range terrains {
		if terrain.Tile.Type != "grass" && terrain.Tile.Type != "dirt" {
			log.Default().Println(fmt.Sprintf("invalid tile type %s at position (%d, %d)", terrain.Tile.Type, terrain.X, terrain.Y))
			return http.StatusBadRequest, gin.H{"error": fmt.Sprintf("building can only be placed on grass or dirt tiles, found %s at position (%d, %d)", terrain.Tile.Type, terrain.X, terrain.Y)}
		}
	}

	// 4. Check if building overlaps with existing buildings
	var existingBuildings []models.PlayerBuilding
	if err := database.DB.Preload("Building").Where("player_id = ? AND map_id = ?", request.PlayerID, request.MapID).Find(&existingBuildings).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to fetch existing buildings for player_id %d on map_id %d", request.PlayerID, request.MapID), err)
		return http.StatusInternalServerError, gin.H{"error": "failed to validate building placement"}
	}

	for _, existingBuilding := range existingBuildings {
		// Check if the new building overlaps with existing building
		newBuildingMaxX := request.X + building.Width - 1
		newBuildingMaxY := request.Y + building.Length - 1
		existingBuildingMaxX := existingBuilding.X + existingBuilding.Building.Width - 1
		existingBuildingMaxY := existingBuilding.Y + existingBuilding.Building.Length - 1

		// Check for overlap: two rectangles overlap if they overlap in both X and Y dimensions
		xOverlap := request.X <= existingBuildingMaxX && newBuildingMaxX >= existingBuilding.X
		yOverlap := request.Y <= existingBuildingMaxY && newBuildingMaxY >= existingBuilding.Y

		if xOverlap && yOverlap {
			log.Default().Println(fmt.Sprintf("building overlaps with existing building at position (%d, %d)", existingBuilding.X, existingBuilding.Y))
			return http.StatusBadRequest, gin.H{"error": "building overlaps with an existing building"}
		}
	}

	// 5. Create the player building
	playerBuilding := models.PlayerBuilding{
		PlayerID:        request.PlayerID,
		BuildingID:      request.BuildingID,
		MapID:           request.MapID,
		BuildingLevelID: buildingLevel.ID,
		X:               request.X,
		Y:               request.Y,
	}

	if err := database.DB.Create(&playerBuilding).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to create player building for player_id %d", request.PlayerID), err)
		return http.StatusInternalServerError, gin.H{"error": "failed to place building"}
	}

	// 6. Load the created building with its associations for the response
	if err := database.DB.Preload("Building").Preload("Building.Category").Preload("BuildingLevel").First(&playerBuilding, playerBuilding.ID).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to load created building with ID %d", playerBuilding.ID), err)
		return http.StatusInternalServerError, gin.H{"error": "failed to load building details"}
	}

	return http.StatusCreated, gin.H{"player_building": playerBuilding.ToDTO()}
}
