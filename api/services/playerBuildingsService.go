package services

import (
	"API/api/dto"
	"API/api/dto/requests"
	"API/api/dto/responses"
	"API/database"
	"API/models"
	"fmt"
	"log"
	"net/http"
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

func validateCoordinates(x, y int) error {
	if x < 0 || y < 0 {
		log.Default().Printf("invalid coordinates: x=%d, y=%d", x, y)
		return fmt.Errorf("coordinates must be non-negative")
	}
	return nil
}

func getBuildingWithLevel(buildingID uint) (*models.Building, *models.BuildingLevel, error) {
	var building models.Building
	if err := database.DB.First(&building, buildingID).Error; err != nil {
		log.Default().Printf("building not found, building_id %d: %s", buildingID, err)
		return nil, nil, fmt.Errorf("building not found")
	}

	var buildingLevel models.BuildingLevel
	if err := database.DB.Where("building_id = ? AND level = ?", buildingID, 1).First(&buildingLevel).Error; err != nil {
		log.Default().Printf("building level 1 not found for building_id %d: %s", buildingID, err)
		return nil, nil, fmt.Errorf("building level not found")
	}

	return &building, &buildingLevel, nil
}

func buildCoordinatesList(startX, startY, width, length int) []struct {
	X int
	Y int
} {
	var coordinates []struct {
		X int
		Y int
	}
	for x := startX; x < startX+width; x++ {
		for y := startY; y < startY+length; y++ {
			coordinates = append(coordinates, struct {
				X int
				Y int
			}{X: x, Y: y})
		}
	}
	return coordinates
}

func validateTerrainForBuilding(mapID uint, x, y, width, length int) error {
	coordinates := buildCoordinatesList(x, y, width, length)

	// Fetch all terrains in one query
	var terrains []models.Terrain
	query := database.DB.Preload("Tile").Where("map_id = ?", mapID)

	// Build OR conditions for all coordinates
	orConditions := database.DB.Where("1 = 0") // Start with false condition
	for _, coord := range coordinates {
		orConditions = orConditions.Or("(x = ? AND y = ?)", coord.X, coord.Y)
	}

	if err := query.Where(orConditions).Find(&terrains).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to fetch terrains for building placement on map_id %d", mapID), err)
		return fmt.Errorf("failed to validate terrain")
	}

	// Verify we found all expected tiles
	if len(terrains) != len(coordinates) {
		log.Default().Println(fmt.Sprintf("not all terrain tiles found for building placement, expected %d, found %d", len(coordinates), len(terrains)))
		return fmt.Errorf("some terrain tiles are missing")
	}

	// Validate all tiles are grass or dirt
	for _, terrain := range terrains {
		if terrain.Tile.Type != "grass" && terrain.Tile.Type != "dirt" {
			log.Default().Println(fmt.Sprintf("invalid tile type %s at position (%d, %d)", terrain.Tile.Type, terrain.X, terrain.Y))
			return fmt.Errorf("building can only be placed on grass or dirt tiles, found %s at position (%d, %d)", terrain.Tile.Type, terrain.X, terrain.Y)
		}
	}

	return nil
}

func checkBuildingOverlap(mapID uint, x, y, width, length int) error {
	var existingBuildings []models.PlayerBuilding
	if err := database.DB.Preload("Building").Where("map_id = ?", mapID).Find(&existingBuildings).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to fetch existing buildings on map_id %d", mapID), err)
		return fmt.Errorf("failed to validate building placement")
	}

	newBuildingMaxX := x + width - 1
	newBuildingMaxY := y + length - 1

	for _, existingBuilding := range existingBuildings {
		existingBuildingMaxX := existingBuilding.X + existingBuilding.Building.Width - 1
		existingBuildingMaxY := existingBuilding.Y + existingBuilding.Building.Length - 1

		// Check for overlap: two rectangles overlap if they overlap in both X and Y dimensions
		xOverlap := x <= existingBuildingMaxX && newBuildingMaxX >= existingBuilding.X
		yOverlap := y <= existingBuildingMaxY && newBuildingMaxY >= existingBuilding.Y

		if xOverlap && yOverlap {
			log.Default().Println(fmt.Sprintf("building overlaps with existing building at position (%d, %d)", existingBuilding.X, existingBuilding.Y))
			return fmt.Errorf("building overlaps with an existing building")
		}
	}

	return nil
}

func createPlayerBuilding(request requests.AddBuildingRequest, buildingLevelID uint) (*models.PlayerBuilding, error) {
	playerBuilding := models.PlayerBuilding{
		PlayerID:        request.PlayerID,
		BuildingID:      request.BuildingID,
		MapID:           request.MapID,
		BuildingLevelID: buildingLevelID,
		X:               request.X,
		Y:               request.Y,
	}

	if err := database.DB.Create(&playerBuilding).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to create player building for player_id %d", request.PlayerID), err)
		return nil, fmt.Errorf("failed to place building")
	}

	return &playerBuilding, nil
}

func loadPlayerBuildingWithAssociations(playerBuilding *models.PlayerBuilding) error {
	if err := database.DB.Preload("Building").Preload("Building.Category").Preload("BuildingLevel").First(playerBuilding, playerBuilding.ID).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to load created building with ID %d", playerBuilding.ID), err)
		return fmt.Errorf("failed to load building details")
	}
	return nil
}

func AddPlayerBuilding(request requests.AddBuildingRequest) (int, responses.AddPlayerBuildingResponse) {
	// Validate coordinates are non-negative
	if err := validateCoordinates(request.X, request.Y); err != nil {
		return http.StatusBadRequest, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	// Get building and its level 1
	building, buildingLevel, err := getBuildingWithLevel(request.BuildingID)
	if err != nil {
		if err.Error() == "building not found" {
			return http.StatusNotFound, responses.AddPlayerBuildingResponse{Error: err.Error()}
		}
		return http.StatusNotFound, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	// Validate terrain tiles are grass or dirt
	if err := validateTerrainForBuilding(request.MapID, request.X, request.Y, building.Width, building.Length); err != nil {
		if err.Error() == "failed to validate terrain" {
			return http.StatusInternalServerError, responses.AddPlayerBuildingResponse{Error: err.Error()}
		}
		return http.StatusBadRequest, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	// Check for overlap with existing buildings
	if err := checkBuildingOverlap(request.MapID, request.X, request.Y, building.Width, building.Length); err != nil {
		if err.Error() == "failed to validate building placement" {
			return http.StatusInternalServerError, responses.AddPlayerBuildingResponse{Error: err.Error()}
		}
		return http.StatusBadRequest, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	// Create the player building
	playerBuilding, err := createPlayerBuilding(request, buildingLevel.ID)
	if err != nil {
		return http.StatusInternalServerError, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	// Load the created building with its associations for the response
	if err := loadPlayerBuildingWithAssociations(playerBuilding); err != nil {
		return http.StatusInternalServerError, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	playerBuildingDTO := playerBuilding.ToDTO()
	return http.StatusCreated, responses.AddPlayerBuildingResponse{PlayerBuilding: &playerBuildingDTO}
}
