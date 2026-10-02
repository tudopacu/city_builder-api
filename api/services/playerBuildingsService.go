package services

import (
	"API/api/dto"
	"API/api/dto/requests"
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

func GetPlayerBuildings(playerId uint, mapId uint) ([]dto.PlayerBuilding, error) {
	playerBuildingIDs := database.DB.
		Model(&models.PlayerBuilding{}).
		Select("id").
		Where("player_id = ? AND map_id = ?", playerId, mapId)

	if err := database.DB.Model(&models.PlayerBuildingProduction{}).
		Where("player_id = ? AND player_building_id IN (?) AND status = ? AND end_time <= ?",
			playerId, playerBuildingIDs, "PENDING", time.Now()).
		Update("status", "DONE").Error; err != nil {
		log.Default().Printf("failed to update expired productions for player_id %d on map_id %d: %s",
			playerId, mapId, err)
		return []dto.PlayerBuilding{}, fmt.Errorf("failed to update expired productions")
	}

	var playerBuildingModels []models.PlayerBuilding

	if err := database.DB.
		Preload("Building").
		Preload("Building.Category").
		Preload("Building.Levels").
		Preload("BuildingLevel").
		Preload("BuildingCurrentProduction", "status <> ?", "COLLECTED").
		Preload("BuildingCurrentProduction.BuildingProduction.Item").
		Find(&playerBuildingModels, "player_id = ? AND map_id = ?", playerId, mapId).
		Error; err != nil {

		log.Default().Printf("failed to fetch player buildings for player_id %d on map_id %d: %s",
			playerId, mapId, err)

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

func getBuildingWithLevel(tx *gorm.DB, buildingID uint) (*models.Building, *models.BuildingLevel, error) {
	var building models.Building
	if err := tx.Preload("Category").First(&building, buildingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("building not found")
		}
		return nil, nil, fmt.Errorf("failed to load building: %w", err)
	}

	var buildingLevel models.BuildingLevel
	if err := tx.Where("building_id = ? AND level = ?", buildingID, 1).First(&buildingLevel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("building level not found")
		}
		return nil, nil, fmt.Errorf("failed to load building level: %w", err)
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

func validateTerrainForBuilding(tx *gorm.DB, mapID uint, x, y, width, length int) error {
	coordinates := buildCoordinatesList(x, y, width, length)

	// Fetch all terrains in one query
	var terrains []models.Terrain
	query := tx.Preload("Tile").Where("map_id = ?", mapID)

	// Build OR conditions for all coordinates
	orConditions := tx.Where("1 = 0") // Start with false condition
	for _, coord := range coordinates {
		orConditions = orConditions.Or("(x = ? AND y = ?)", coord.X, coord.Y)
	}

	if err := query.Where(orConditions).Find(&terrains).Error; err != nil {
		log.Default().Printf("failed to fetch terrains for building placement on map_id %d: %s", mapID, err)
		return fmt.Errorf("failed to validate terrain")
	}

	// Verify we found all expected tiles
	if len(terrains) != len(coordinates) {
		log.Default().Printf("not all terrain tiles found for building placement, expected %d, found %d", len(coordinates), len(terrains))
		return fmt.Errorf("some terrain tiles are missing")
	}

	// Validate all tiles are grass or dirt
	for _, terrain := range terrains {
		if terrain.Tile.Type != "grass" && terrain.Tile.Type != "dirt" {
			log.Default().Printf("invalid tile type %s at position (%d, %d)", terrain.Tile.Type, terrain.X, terrain.Y)
			return fmt.Errorf("building can only be placed on grass or dirt tiles, found %s at position (%d, %d)", terrain.Tile.Type, terrain.X, terrain.Y)
		}
	}

	return nil
}

func checkBuildingOverlap(tx *gorm.DB, mapID uint, x, y, width, length int) error {
	var existingBuildings []models.PlayerBuilding
	if err := tx.Preload("Building").Where("map_id = ?", mapID).Find(&existingBuildings).Error; err != nil {
		log.Default().Printf("failed to fetch existing buildings on map_id %d: %s", mapID, err)
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
			log.Default().Printf("building overlaps with existing building at position (%d, %d)", existingBuilding.X, existingBuilding.Y)
			return fmt.Errorf("building overlaps with an existing building")
		}
	}

	return nil
}

func createPlayerBuilding(tx *gorm.DB, request requests.AddBuildingRequest, buildingLevelID uint) (*models.PlayerBuilding, error) {
	playerBuilding := models.PlayerBuilding{
		PlayerID:        request.PlayerID,
		BuildingID:      request.BuildingID,
		MapID:           request.MapID,
		BuildingLevelID: buildingLevelID,
		X:               request.X,
		Y:               request.Y,
	}

	if err := tx.Create(&playerBuilding).Error; err != nil {
		log.Default().Printf("failed to create player building for player_id %d: %s", request.PlayerID, err)
		return nil, fmt.Errorf("failed to place building")
	}

	return &playerBuilding, nil
}

func loadPlayerBuildingWithAssociations(tx *gorm.DB, playerBuilding *models.PlayerBuilding) error {
	if err := tx.Preload("Building").Preload("Building.Category").Preload("BuildingLevel").First(playerBuilding, playerBuilding.ID).Error; err != nil {
		log.Default().Printf("failed to load created building with ID %d: %s", playerBuilding.ID, err)
		return fmt.Errorf("failed to load building details")
	}
	return nil
}

func DeletePlayerBuilding(playerBuildingID uint) (int, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var playerBuilding models.PlayerBuilding
		if err := tx.Preload("Building.Category").First(&playerBuilding, playerBuildingID).Error; err != nil {
			return err
		}

		if playerBuilding.Building.Category.Name == "Storage" {
			var inventories []models.PlayerInventory
			if err := tx.Where("player_building_id = ?", playerBuilding.ID).Find(&inventories).Error; err != nil {
				return fmt.Errorf("failed to load storage inventories: %w", err)
			}
			for _, inventory := range inventories {
				if err := tx.Where("player_inventory_id = ?", inventory.ID).Delete(&models.PlayerInventoryItem{}).Error; err != nil {
					return fmt.Errorf("failed to delete storage inventory items: %w", err)
				}
			}
			if err := tx.Where("player_building_id = ?", playerBuilding.ID).Delete(&models.PlayerInventory{}).Error; err != nil {
				return fmt.Errorf("failed to delete storage inventories: %w", err)
			}
		}
		return tx.Delete(&playerBuilding).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Default().Printf("player building not found, id %d: %s", playerBuildingID, err)
			return http.StatusNotFound, fmt.Errorf("player building not found")
		}
		log.Default().Printf("failed to delete player building id %d: %s", playerBuildingID, err)
		return http.StatusInternalServerError, fmt.Errorf("failed to delete player building")
	}

	return http.StatusOK, nil
}

func AddPlayerBuilding(request requests.AddBuildingRequest) (int, responses.AddPlayerBuildingResponse) {
	// Validate coordinates are non-negative
	if err := validateCoordinates(request.X, request.Y); err != nil {
		return http.StatusBadRequest, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	statusCode := http.StatusInternalServerError
	var playerBuilding *models.PlayerBuilding
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		building, buildingLevel, err := getBuildingWithLevel(tx, request.BuildingID)
		if err != nil {
			if err.Error() == "building not found" || err.Error() == "building level not found" {
				statusCode = http.StatusNotFound
			}
			return err
		}

		if err := validateTerrainForBuilding(tx, request.MapID, request.X, request.Y, building.Width, building.Length); err != nil {
			if err.Error() != "failed to validate terrain" {
				statusCode = http.StatusBadRequest
			}
			return err
		}

		if err := checkBuildingOverlap(tx, request.MapID, request.X, request.Y, building.Width, building.Length); err != nil {
			if err.Error() != "failed to validate building placement" {
				statusCode = http.StatusBadRequest
			}
			return err
		}

		if costStatus, err := consumeBuildingConstructionCosts(tx, request.PlayerID, request.BuildingID); err != nil {
			statusCode = costStatus
			return err
		}

		playerBuilding, err = createPlayerBuilding(tx, request, buildingLevel.ID)
		if err != nil {
			return err
		}
		if building.Category.Name == "Storage" {
			inventory := models.PlayerInventory{
				PlayerID:         request.PlayerID,
				MapID:            request.MapID,
				PlayerBuildingID: playerBuilding.ID,
				Capacity:         buildingLevel.Capacity,
			}
			if err := tx.Create(&inventory).Error; err != nil {
				return fmt.Errorf("failed to create storage inventory: %w", err)
			}
		}
		return loadPlayerBuildingWithAssociations(tx, playerBuilding)
	})
	if err != nil {
		if statusCode == http.StatusInternalServerError {
			log.Default().Printf("failed to add building_id %d for player_id %d: %s", request.BuildingID, request.PlayerID, err)
			return statusCode, responses.AddPlayerBuildingResponse{Error: "failed to place building"}
		}
		return statusCode, responses.AddPlayerBuildingResponse{Error: err.Error()}
	}

	playerBuildingDTO := playerBuilding.ToDTO()
	return http.StatusCreated, responses.AddPlayerBuildingResponse{PlayerBuilding: &playerBuildingDTO}
}

func consumeBuildingConstructionCosts(tx *gorm.DB, playerID, buildingID uint) (int, error) {
	var costs []models.BuildingConstructionCost
	if err := tx.Where("building_id = ?", buildingID).Order("item_id ASC").Order("id ASC").Find(&costs).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to load construction costs: %w", err)
	}

	required := make(map[uint]int)
	var itemIDs []uint
	for _, cost := range costs {
		if cost.Quantity < 1 {
			return http.StatusInternalServerError, fmt.Errorf("invalid construction cost for item %d", cost.ItemID)
		}
		if _, exists := required[cost.ItemID]; !exists {
			itemIDs = append(itemIDs, cost.ItemID)
		}
		required[cost.ItemID] += cost.Quantity
	}
	if len(required) == 0 {
		return http.StatusOK, nil
	}

	// Use the same lock order as production collection, including locking reads
	// of item quantities so concurrent construction cannot spend the same stock.
	var inventories []models.PlayerInventory
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("InventoryItems", func(db *gorm.DB) *gorm.DB {
			return db.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id ASC")
		}).
		Where("player_id = ?", playerID).
		Order("id ASC").Find(&inventories).Error; err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to load player inventories: %w", err)
	}

	available := make(map[uint]int)
	for _, inventory := range inventories {
		for _, item := range inventory.InventoryItems {
			available[item.ItemID] += max(0, item.Quantity)
		}
	}
	// Check every cost before modifying any inventory.
	for _, itemID := range itemIDs {
		if available[itemID] < required[itemID] {
			return http.StatusBadRequest, fmt.Errorf("not enough items for construction: item %d, required %d, available %d",
				itemID, required[itemID], available[itemID])
		}
	}

	for _, inventory := range inventories {
		for _, item := range inventory.InventoryItems {
			quantity := min(required[item.ItemID], item.Quantity)
			if quantity <= 0 {
				continue
			}
			result := tx.Model(&item).Where("quantity >= ?", quantity).
				Update("quantity", gorm.Expr("quantity - ?", quantity))
			if result.Error != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to consume construction item: %w", result.Error)
			}
			if result.RowsAffected != 1 {
				return http.StatusInternalServerError, fmt.Errorf("construction item %d changed during placement", item.ID)
			}
			required[item.ItemID] -= quantity
		}
	}
	return http.StatusOK, nil
}
