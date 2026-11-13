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
	itemsCacheKey = "items:all"
	itemsCacheTTL = 10 * time.Minute
)

func GetAllItems(ctx context.Context) ([]dto.Item, error) {
	return cached(ctx, itemsCacheKey, itemsCacheTTL, loadItems)
}

func loadItems(ctx context.Context) ([]dto.Item, error) {
	var items []models.Item
	if err := database.DB.WithContext(ctx).Order("id ASC").Find(&items).Error; err != nil {
		log.Default().Println("failed to fetch items", err)
		return nil, fmt.Errorf("failed to fetch items")
	}

	itemDTOs := make([]dto.Item, 0, len(items))
	for _, item := range items {
		itemDTOs = append(itemDTOs, item.ToDTO())
	}

	return itemDTOs, nil
}
