package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

const (
	tilesCacheKey   = "tiles:all"
	mapsAllCacheKey = "maps:all"
	mapCacheTTL     = 10 * time.Minute
)

var ErrMapNotFound = errors.New("map not found")

func GetAllMaps(ctx context.Context) ([]dto.Map, error) {
	return cached(ctx, mapsAllCacheKey, mapCacheTTL, func(ctx context.Context) ([]dto.Map, error) {
		var maps []models.Map
		if err := database.DB.WithContext(ctx).Preload("Terrains.Tile").Order("id ASC").Find(&maps).Error; err != nil {
			log.Default().Println("failed to fetch maps", err)
			return nil, fmt.Errorf("failed to fetch maps")
		}

		dtoMaps := make([]dto.Map, len(maps))
		for i, m := range maps {
			dtoMaps[i] = m.ToDTO()
		}
		return dtoMaps, nil
	})
}

func GetTiles(ctx context.Context) ([]dto.Tile, error) {
	return cached(ctx, tilesCacheKey, mapCacheTTL, func(ctx context.Context) ([]dto.Tile, error) {
		var tiles []models.Tile
		if err := database.DB.WithContext(ctx).Order("id ASC").Find(&tiles).Error; err != nil {
			log.Default().Println("failed to fetch tiles", err)
			return nil, fmt.Errorf("failed to fetch tiles")
		}

		dtoTiles := make([]dto.Tile, 0, len(tiles))
		for _, tile := range tiles {
			dtoTiles = append(dtoTiles, tile.ToDTO())
		}
		return dtoTiles, nil
	})
}

func GetMapByID(ctx context.Context, mapId uint) (*dto.Map, error) {
	cacheKey := fmt.Sprintf("map:%d", mapId)

	// A missing map is cached as nil, so repeated lookups of unknown IDs are
	// answered from the cache instead of reaching the database every time.
	mapDto, err := cached(ctx, cacheKey, mapCacheTTL, func(ctx context.Context) (*dto.Map, error) {
		var mapModel models.Map
		if err := database.DB.WithContext(ctx).Preload("Terrains.Tile").First(&mapModel, mapId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			log.Default().Println(fmt.Sprintf("failed to fetch map, map_id %d", mapId), err)
			return nil, fmt.Errorf("failed to fetch map, map_id %d", mapId)
		}

		dtoMap := mapModel.ToDTO()
		return &dtoMap, nil
	})
	if err != nil {
		return nil, err
	}
	if mapDto == nil {
		return nil, ErrMapNotFound
	}
	return mapDto, nil
}
