package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	redisClient "API/redis"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const (
	tilesCacheKey = "tiles:all"
	mapsAllCacheKey = "maps:all"
	mapCacheTTL   = 10 * time.Minute
)

func GetAllMaps() ([]dto.Map, error) {
	ctx := context.Background()

	cached, err := redisClient.RDB.Get(ctx, mapsAllCacheKey).Result()
	if err == nil {
		var dtoMaps []dto.Map
		if jsonErr := json.Unmarshal([]byte(cached), &dtoMaps); jsonErr == nil {
			return dtoMaps, nil
		}
	}

	var maps []models.Map
	if err := database.DB.Preload("Terrains.Tile").Find(&maps).Error; err != nil {
		log.Default().Println("failed to fetch maps", err)
		return nil, fmt.Errorf("failed to fetch maps")
	}

	dtoMaps := make([]dto.Map, len(maps))
	for i, m := range maps {
		dtoMaps[i] = m.ToDTO()
	}

	if data, jsonErr := json.Marshal(dtoMaps); jsonErr == nil {
		redisClient.RDB.Set(ctx, mapsAllCacheKey, data, mapCacheTTL)
	}

	return dtoMaps, nil
}

func GetTiles() ([]dto.Tile, error) {
	ctx := context.Background()

	cached, err := redisClient.RDB.Get(ctx, tilesCacheKey).Result()
	if err == nil {
		var dtoTiles []dto.Tile
		if jsonErr := json.Unmarshal([]byte(cached), &dtoTiles); jsonErr == nil {
			return dtoTiles, nil
		}
	}

	var tiles []models.Tile

	if err := database.DB.Find(&tiles).Error; err != nil {
		log.Default().Println("failed to fetch tiles", err)
		return nil, fmt.Errorf("failed to fetch tiles")
	}

	var dtoTiles []dto.Tile
	for _, tile := range tiles {
		dtoTiles = append(dtoTiles, tile.ToDTO())
	}

	if data, jsonErr := json.Marshal(dtoTiles); jsonErr == nil {
		redisClient.RDB.Set(ctx, tilesCacheKey, data, mapCacheTTL)
	}

	return dtoTiles, nil
}

func GetMapByID(mapId uint) (*dto.Map, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("map:%d", mapId)

	cached, err := redisClient.RDB.Get(ctx, cacheKey).Result()
	if err == nil {
		var dtoMap dto.Map
		if jsonErr := json.Unmarshal([]byte(cached), &dtoMap); jsonErr == nil {
			return &dtoMap, nil
		}
	}

	var mapModel models.Map

	if err := database.DB.Preload("Terrains.Tile").First(&mapModel, mapId).Error; err != nil {
		log.Default().Println(fmt.Sprintf("failed to fetch map, map_id %d", mapId), err)
		return nil, fmt.Errorf("failed to fetch map, map_id %d", mapId)
	}

	dtoMap := mapModel.ToDTO()

	if data, jsonErr := json.Marshal(dtoMap); jsonErr == nil {
		redisClient.RDB.Set(ctx, cacheKey, data, mapCacheTTL)
	}

	return &dtoMap, nil
}
