package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

func GetTiles() ([]dto.Tile, error) {
	var tiles []models.Tile

	if err := database.DB.Find(&tiles).Error; err != nil {
		log.Default().Println("failed to fetch tiles", err)
		return nil, fmt.Errorf("failed to fetch tiles")
	}

	var dtoTiles []dto.Tile
	for _, tile := range tiles {
		dtoTiles = append(dtoTiles, tile.ToDTO())
	}

	return dtoTiles, nil
}

func GetMapByID(mapId uint) (*dto.Map, error) {
	var mapModel models.Map

	if err := database.DB.Preload("Terrains.Tile").First(&mapModel, mapId).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to fetch map, map_id %d", mapId), err)
		return nil, fmt.Errorf("failed to fetch map, map_id %d", mapId)
	}

	dtoMap := mapModel.ToDTO()
	return &dtoMap, nil
}
