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

const newsCacheTTL = 5 * time.Minute

type newsCacheEntry struct {
	News       []dto.News `json:"news"`
	TotalCount int64      `json:"total_count"`
}

func GetNews(page, pageSize int) ([]dto.News, int64, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("news:page:%d:size:%d", page, pageSize)

	cached, err := redisClient.RDB.Get(ctx, cacheKey).Result()
	if err == nil {
		var entry newsCacheEntry
		if jsonErr := json.Unmarshal([]byte(cached), &entry); jsonErr == nil {
			return entry.News, entry.TotalCount, nil
		}
	}

	var newsModels []models.News
	var totalCount int64

	// Get total count
	if err := database.DB.Model(&models.News{}).Count(&totalCount).Error; err != nil {
		log.Default().Println("failed to count news", err)
		return nil, 0, fmt.Errorf("failed to count news")
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated news ordered by created_at descending (most recent first)
	if err := database.DB.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&newsModels).Error; err != nil {
		log.Default().Println("failed to fetch news", err)
		return nil, 0, fmt.Errorf("failed to fetch news")
	}

	var dtoNews []dto.News
	for _, news := range newsModels {
		dtoNews = append(dtoNews, news.ToDTO())
	}

	entry := newsCacheEntry{News: dtoNews, TotalCount: totalCount}
	if data, jsonErr := json.Marshal(entry); jsonErr == nil {
		redisClient.RDB.Set(ctx, cacheKey, data, newsCacheTTL)
	}

	return dtoNews, totalCount, nil
}
