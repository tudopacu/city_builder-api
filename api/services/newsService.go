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

const newsCacheTTL = 5 * time.Minute

type newsCacheEntry struct {
	News       []dto.News `json:"news"`
	TotalCount int64      `json:"total_count"`
}

func GetNews(ctx context.Context, page, pageSize int) ([]dto.News, int64, error) {
	cacheKey := fmt.Sprintf("news:page:%d:size:%d", page, pageSize)

	entry, err := cached(ctx, cacheKey, newsCacheTTL, func() (newsCacheEntry, error) {
		var newsModels []models.News
		var totalCount int64

		// Get total count
		if err := database.DB.Model(&models.News{}).Count(&totalCount).Error; err != nil {
			log.Default().Println("failed to count news", err)
			return newsCacheEntry{}, fmt.Errorf("failed to count news")
		}

		// Calculate offset
		offset := (page - 1) * pageSize

		// Get paginated news ordered by created_at descending (most recent first)
		if err := database.DB.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&newsModels).Error; err != nil {
			log.Default().Println("failed to fetch news", err)
			return newsCacheEntry{}, fmt.Errorf("failed to fetch news")
		}

		dtoNews := make([]dto.News, 0, len(newsModels))
		for _, news := range newsModels {
			dtoNews = append(dtoNews, news.ToDTO())
		}

		return newsCacheEntry{News: dtoNews, TotalCount: totalCount}, nil
	})
	if err != nil {
		return nil, 0, err
	}

	return entry.News, entry.TotalCount, nil
}
