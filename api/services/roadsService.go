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
	roadsCacheKey = "roads:all"
	roadsCacheTTL = 10 * time.Minute
)

func GetAllRoads() ([]dto.Road, error) {
	ctx := context.Background()

	cached, err := redisClient.RDB.Get(ctx, roadsCacheKey).Result()
	if err == nil {
		var roads []dto.Road
		if jsonErr := json.Unmarshal([]byte(cached), &roads); jsonErr == nil {
			return roads, nil
		}
	}

	var roads []models.Road
	if err := database.DB.Find(&roads).Error; err != nil {
		log.Default().Println("failed to fetch roads", err)
		return nil, fmt.Errorf("failed to fetch roads")
	}

	var roadDTOs []dto.Road
	for _, road := range roads {
		roadDTOs = append(roadDTOs, road.ToDTO())
	}

	if data, jsonErr := json.Marshal(roadDTOs); jsonErr == nil {
		redisClient.RDB.Set(ctx, roadsCacheKey, data, roadsCacheTTL)
	}

	return roadDTOs, nil
}
