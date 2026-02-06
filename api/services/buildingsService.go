package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

func GetAllBuildings() ([]dto.BuildingWithDetails, error) {
	var buildings []models.Building

	// Preload related data: Category, ConstructionCosts with Item, and Productions with Item
	if err := database.DB.
		Preload("Category").
		Preload("ConstructionCosts.Item").
		Preload("Productions.Item").
		Find(&buildings).Error; err != nil {
		log.Default().Println("failed to fetch buildings", err)
		return nil, fmt.Errorf("failed to fetch buildings")
	}

	var buildingsWithDetails []dto.BuildingWithDetails
	for _, building := range buildings {
		// Map construction costs
		var costs []dto.BuildingCost
		for _, cost := range building.ConstructionCosts {
			costs = append(costs, dto.BuildingCost{
				ItemID:   cost.ItemID,
				ItemName: cost.Item.Name,
				Quantity: cost.Quantity,
			})
		}

		// Map production items
		var productions []dto.BuildingProductionItem
		for _, prod := range building.Productions {
			productions = append(productions, dto.BuildingProductionItem{
				ItemID:                prod.ItemID,
				ItemName:              prod.Item.Name,
				Quantity:              prod.Quantity,
				ProductionTimeSeconds: prod.ProductionTimeSeconds,
			})
		}

		buildingsWithDetails = append(buildingsWithDetails, dto.BuildingWithDetails{
			ID:               building.ID,
			Name:             building.Name,
			ImageURL:         building.ImageURL,
			Description:      building.Description,
			Width:            building.Width,
			Length:           building.Length,
			BuildingCategory: building.Category.Name,
			Costs:            costs,
			Productions:      productions,
		})
	}

	return buildingsWithDetails, nil
}
