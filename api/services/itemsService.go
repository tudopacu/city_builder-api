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
	itemsCacheKey = "items:all"
	itemsCacheTTL = 10 * time.Minute
)

func GetAllItems() ([]dto.Item, error) {
	ctx := context.Background()

	cached, err := redisClient.RDB.Get(ctx, itemsCacheKey).Result()
	if err == nil {
		var items []dto.Item
		if jsonErr := json.Unmarshal([]byte(cached), &items); jsonErr == nil {
			return items, nil
		}
	}

	var items []models.Item
	if err := database.DB.Find(&items).Error; err != nil {
		log.Default().Println("failed to fetch items", err)
		return nil, fmt.Errorf("failed to fetch items")
	}

	var itemDTOs []dto.Item
	for _, item := range items {
		itemDTOs = append(itemDTOs, item.ToDTO())
	}

	if data, jsonErr := json.Marshal(itemDTOs); jsonErr == nil {
		redisClient.RDB.Set(ctx, itemsCacheKey, data, itemsCacheTTL)
	}

	return itemDTOs, nil
}
