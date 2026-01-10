package services

import (
	"API/api/dto"
	"API/database"
	"API/models"
	"fmt"
	"log"
)

func GetNews(page, pageSize int) ([]dto.News, int64, error) {
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

	return dtoNews, totalCount, nil
}
