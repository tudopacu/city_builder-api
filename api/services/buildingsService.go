package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"context"
	"fmt"
	"log"
	"time"
)

const (
	buildingsCacheKey = "buildings:all"
	buildingsCacheTTL = 10 * time.Minute
)

func GetAllBuildings(ctx context.Context) ([]dto.BuildingWithDetails, error) {
	return cached(ctx, buildingsCacheKey, buildingsCacheTTL, loadBuildings)
}

func loadBuildings() ([]dto.BuildingWithDetails, error) {
	var buildings []models.Building

	// New buildings are always placed at level 1, so only that level is needed.
	if err := database.DB.
		Preload("Category").
		Preload("ConstructionCosts.Item").
		Preload("Productions.Item").
		Preload("Levels", "level = ?", 1).
		Order("id ASC").
		Find(&buildings).Error; err != nil {
		log.Default().Println("failed to fetch buildings", err)
		return nil, fmt.Errorf("failed to fetch buildings")
	}

	buildingsWithDetails := make([]dto.BuildingWithDetails, 0, len(buildings))
	for _, building := range buildings {
		// AddPlayerBuilding rejects buildings without a level 1 row, so listing
		// them would offer the player something that can never be placed.
		if len(building.Levels) == 0 {
			log.Default().Printf("building %d has no level 1, leaving it out of the catalog", building.ID)
			continue
		}
		level := building.Levels[0]

		costs := make([]dto.BuildingCost, 0, len(building.ConstructionCosts))
		for _, cost := range building.ConstructionCosts {
			costs = append(costs, dto.BuildingCost{
				ItemID:   cost.ItemID,
				ItemName: cost.Item.Name,
				Quantity: cost.Quantity,
			})
		}

		productions := make([]dto.BuildingProductionItem, 0, len(building.Productions))
		for _, prod := range building.Productions {
			productions = append(productions, dto.BuildingProductionItem{
				BuildingProductionID:  prod.ID,
				ItemID:                prod.ItemID,
				ItemName:              prod.Item.Name,
				Quantity:              prod.Quantity,
				ProductionTimeSeconds: prod.ProductionTimeSeconds,
			})
		}

		buildingsWithDetails = append(buildingsWithDetails, dto.BuildingWithDetails{
			ID:                 building.ID,
			Name:               building.Name,
			ImageURL:           building.ImageURL,
			Description:        building.Description,
			Width:              building.Width,
			Length:             building.Length,
			BuildingCategory:   building.Category.Name,
			PopulationCapacity: level.PopulationCapacity,
			PopulationCost:     level.PopulationCost,
			Costs:              costs,
			Productions:        productions,
		})
	}

	return buildingsWithDetails, nil
}
