package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
)

func GetTiles() ([]dto.Tile, error) {
	var tiles []models.Tile

	if err := database.DB.Find(&tiles).Error; err != nil {
		return nil, err
	}

	var dtoTiles []dto.Tile
	for _, tile := range tiles {
		dtoTiles = append(dtoTiles, tile.ToDTO())
	}

	return dtoTiles, nil
}
